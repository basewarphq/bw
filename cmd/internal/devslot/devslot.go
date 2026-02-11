package devslot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/cockroachdb/errors"
)

var (
	ErrSlotTaken      = errors.New("slot is already claimed")
	ErrSlotNotClaimed = errors.New("slot is not claimed")
	ErrNoFreeSlots    = errors.New("no free dev slots available")
	ErrTokenMismatch  = errors.New("token does not match")
	ErrNoClaim        = errors.New("no active claim")
)

const (
	keyPrefix     = "dev-slots/"
	claimFileName = "bw.claim"
)

type ClaimFile struct {
	Slot  string `json:"slot"`
	Token string `json:"token"`
}

type LockInfo struct {
	Token     string `json:"token"`
	Label     string `json:"label"`
	ClaimedAt string `json:"claimed_at"`
	LastUsed  string `json:"last_used"`
}

type Store struct {
	Bucket string
	Region string
}

func NewStore(bucket, region string) *Store {
	return &Store{Bucket: bucket, Region: region}
}

func (s *Store) Claim(ctx context.Context, slot, token, label string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	lock := LockInfo{
		Token:     token,
		Label:     label,
		ClaimedAt: now,
		LastUsed:  now,
	}
	body, err := json.Marshal(lock)
	if err != nil {
		return errors.Wrap(err, "marshaling lock info")
	}

	out, err := s.putObject(ctx, keyPrefix+slot+".lock", body, true)
	if err != nil {
		if strings.Contains(out, "PreconditionFailed") ||
			strings.Contains(out, "At least one of the pre-conditions") {
			return errors.Mark(
				errors.Newf("slot %s is already claimed", slot),
				ErrSlotTaken,
			)
		}
		return errors.Newf("claiming slot %s: %s\n%s", slot, err, out)
	}
	return nil
}

func (s *Store) Release(ctx context.Context, slot, token string) error {
	lock, err := s.GetLock(ctx, slot)
	if err != nil {
		return err
	}
	if lock == nil {
		return errors.Mark(
			errors.Newf("slot %s is not claimed", slot),
			ErrSlotNotClaimed,
		)
	}
	if lock.Token != token {
		return errors.Mark(
			errors.Newf("slot %s is claimed by someone else", slot),
			ErrTokenMismatch,
		)
	}

	return s.deleteLock(ctx, slot)
}

func (s *Store) ForceRelease(ctx context.Context, slot string) error {
	lock, err := s.GetLock(ctx, slot)
	if err != nil {
		return err
	}
	if lock == nil {
		return errors.Mark(
			errors.Newf("slot %s is not claimed", slot),
			ErrSlotNotClaimed,
		)
	}

	return s.deleteLock(ctx, slot)
}

func (s *Store) Touch(ctx context.Context, slot, token string) error {
	lock, err := s.GetLock(ctx, slot)
	if err != nil {
		return err
	}
	if lock == nil || lock.Token != token {
		return nil
	}

	lock.LastUsed = time.Now().UTC().Format(time.RFC3339)
	body, err := json.Marshal(lock)
	if err != nil {
		return errors.Wrap(err, "marshaling lock info")
	}

	out, err := s.putObject(ctx, keyPrefix+slot+".lock", body, false)
	if err != nil {
		return errors.Newf("touching slot %s: %s\n%s", slot, err, out)
	}
	return nil
}

func (s *Store) GetLock(ctx context.Context, slot string) (*LockInfo, error) {
	key := keyPrefix + slot + ".lock"

	tmpFile, err := os.CreateTemp("", "devslot-*.json")
	if err != nil {
		return nil, errors.Wrap(err, "creating temp file")
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	_, err = cmdexec.Output(ctx, "/", "aws", "s3api", "get-object",
		"--bucket", s.Bucket,
		"--key", key,
		"--region", s.Region,
		"--no-cli-pager",
		tmpPath,
	)
	if err != nil {
		return nil, nil //nolint:nilnil // nil means "not claimed"
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading lock file for slot %s", slot)
	}

	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, errors.Wrapf(err, "parsing lock for slot %s", slot)
	}
	return &info, nil
}

func (s *Store) ListAll(ctx context.Context, slots []string) (map[string]*LockInfo, error) {
	result := make(map[string]*LockInfo, len(slots))
	for _, slot := range slots {
		info, err := s.GetLock(ctx, slot)
		if err != nil {
			return nil, err
		}
		result[slot] = info
	}
	return result, nil
}

func (s *Store) deleteLock(ctx context.Context, slot string) error {
	_, err := cmdexec.Output(ctx, "/", "aws", "s3api", "delete-object",
		"--bucket", s.Bucket,
		"--key", keyPrefix+slot+".lock",
		"--region", s.Region,
		"--no-cli-pager",
	)
	if err != nil {
		return errors.Wrapf(err, "deleting lock for slot %s", slot)
	}
	return nil
}

func (s *Store) putObject(
	ctx context.Context, key string, body []byte, ifNoneMatch bool,
) (string, error) {
	tmpFile, err := os.CreateTemp("", "devslot-body-*.json")
	if err != nil {
		return "", errors.Wrap(err, "creating temp file")
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(body); err != nil {
		tmpFile.Close()
		return "", errors.Wrap(err, "writing temp file")
	}
	tmpFile.Close()

	args := []string{
		"s3api", "put-object",
		"--bucket", s.Bucket,
		"--key", key,
		"--body", tmpPath,
		"--region", s.Region,
		"--no-cli-pager",
	}
	if ifNoneMatch {
		args = append(args, "--if-none-match", "*")
	}

	out, err := cmdexec.Output(ctx, "/", "aws", args...)
	if err != nil {
		var cmdErr *cmdexec.Error
		if errors.As(err, &cmdErr) {
			return cmdErr.Stderr, err
		}
		return "", err
	}
	return out, nil
}

func EnsureClaim(ctx context.Context, dir, profile string) (*ClaimFile, error) {
	claim, err := ReadClaimFile(dir)
	if err != nil && !errors.Is(err, ErrNoClaim) {
		return nil, err
	}
	if claim != nil {
		TouchClaim(ctx, dir, profile, claim)
		return claim, nil
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return nil, err
	}

	slots := cctx.DevSlots()
	if len(slots) == 0 {
		return nil, errors.New("no Dev* deployments defined in cdk.context.json")
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	accountID, err := AccountID(ctx, profile)
	if err != nil {
		return nil, err
	}

	store := NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)
	label := DefaultLabel(ctx)

	slot, err := ClaimFirstAvailable(ctx, store, slots, token, label)
	if err != nil {
		return nil, err
	}

	claim = &ClaimFile{Slot: slot, Token: token}
	if err := WriteClaimFile(dir, claim); err != nil {
		return nil, err
	}
	return claim, nil
}

func TouchClaim(ctx context.Context, dir, profile string, claim *ClaimFile) {
	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return
	}
	accountID, err := AccountID(ctx, profile)
	if err != nil {
		return
	}
	store := NewStore(cctx.BootstrapBucket(accountID), cctx.PrimaryRegion)
	_ = store.Touch(ctx, claim.Slot, claim.Token)
}

func ClaimFirstAvailable(
	ctx context.Context, store *Store, slots []string, token, label string,
) (string, error) {
	if len(slots) == 0 {
		return "", errors.New("no dev slots defined in cdk.context.json")
	}

	for _, slot := range slots {
		err := store.Claim(ctx, slot, token, label)
		if err == nil {
			return slot, nil
		}
		if !errors.Is(err, ErrSlotTaken) {
			return "", err
		}
	}
	return "", errors.Mark(
		errors.Newf("no free dev slots available: tried %s",
			strings.Join(slots, ", ")),
		ErrNoFreeSlots,
	)
}

func GenerateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Wrap(err, "generating token")
	}
	return hex.EncodeToString(b), nil
}

func DefaultLabel(ctx context.Context) string {
	user := runGit(ctx, "config", "user.name")
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "unknown"
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return user + "@" + host
}

func ClaimFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, claimFileName)
}

func ReadClaimFile(projectRoot string) (*ClaimFile, error) {
	path := ClaimFilePath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Mark(
				errors.New("no active claim — run 'bw infra slots claim' first"),
				ErrNoClaim,
			)
		}
		return nil, errors.Wrapf(err, "reading %s", path)
	}

	var claim ClaimFile
	if err := json.Unmarshal(data, &claim); err != nil {
		return nil, errors.Wrapf(err, "parsing %s", path)
	}
	return &claim, nil
}

func WriteClaimFile(projectRoot string, claim *ClaimFile) error {
	data, err := json.MarshalIndent(claim, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshaling claim file")
	}
	data = append(data, '\n')

	path := ClaimFilePath(projectRoot)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return errors.Wrapf(err, "writing %s", path)
	}
	return nil
}

func RemoveClaimFile(projectRoot string) error {
	path := ClaimFilePath(projectRoot)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "removing %s", path)
	}
	return nil
}

func AccountID(ctx context.Context, profile string) (string, error) {
	args := []string{
		"sts", "get-caller-identity",
		"--query", "Account",
		"--output", "text",
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := cmdexec.Output(ctx, "/", "aws", args...)
	if err != nil {
		return "", errors.Wrap(err, "getting AWS account ID")
	}
	return strings.TrimSpace(out), nil
}

func runGit(ctx context.Context, args ...string) string {
	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
