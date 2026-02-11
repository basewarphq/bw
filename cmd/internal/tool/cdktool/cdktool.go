package cdktool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/basewarphq/bw/cmd/internal/cdkctx"
	"github.com/basewarphq/bw/cmd/internal/cfndeploy"
	"github.com/basewarphq/bw/cmd/internal/cfnparams"
	"github.com/basewarphq/bw/cmd/internal/cfnpatch"
	"github.com/basewarphq/bw/cmd/internal/cfnread"
	"github.com/basewarphq/bw/cmd/internal/cfnvalidate"
	"github.com/basewarphq/bw/cmd/internal/cmdexec"
	"github.com/basewarphq/bw/cmd/internal/devslot"
	"github.com/basewarphq/bw/cmd/internal/devstrategy"
	"github.com/basewarphq/bw/cmd/internal/tool"
	"github.com/cockroachdb/errors"
)

const devSlotExpirationDays = 7

type cdkConfig struct {
	Profile         string              `toml:"profile"`
	DevStrategy     string              `toml:"dev-strategy"`
	LegacyBootstrap bool                `toml:"legacy-bootstrap"`
	PreBootstrap    *preBootstrapConfig `toml:"pre-bootstrap"`
}

type preBootstrapConfig struct {
	Template   string            `toml:"template"`
	Parameters map[string]string `toml:"parameters"`
}

func (c *cdkConfig) cdkArgs(qualifier string) []string {
	var args []string
	if c.LegacyBootstrap {
		args = append(args,
			"--qualifier", qualifier,
			"--toolkit-stack-name", qualifier+"Bootstrap",
		)
	}
	if c.Profile != "" {
		args = append(args, "--profile", c.Profile)
	}
	return args
}

type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Name() string        { return "cdk" }
func (t *Tool) RunsAfter() []string { return nil }

func (t *Tool) DecodeConfig(meta toml.MetaData, raw toml.Primitive) (any, error) {
	var cfg cdkConfig
	if err := meta.PrimitiveDecode(raw, &cfg); err != nil {
		return nil, errors.Wrap(err, "decoding cdk config")
	}
	if cfg.DevStrategy != "" && cfg.DevStrategy != "iam-username" {
		return nil, errors.Newf("dev-strategy must be %q, got %q", "iam-username", cfg.DevStrategy)
	}
	if pb := cfg.PreBootstrap; pb != nil {
		if pb.Template == "" {
			return nil, errors.New("pre-bootstrap.template is required")
		}
		if filepath.IsAbs(pb.Template) {
			return nil, errors.Newf("pre-bootstrap.template must be relative, got %q", pb.Template)
		}
	}
	return cfg, nil
}

func (t *Tool) RequiredBinaries() []tool.BinaryRequirement {
	return []tool.BinaryRequirement{
		{Name: "cdk", Reason: "deploy and manage CDK stacks"},
		{Name: "aws", Reason: "interact with AWS services"},
	}
}

func (t *Tool) RequiredFiles() []tool.FileRequirement {
	return []tool.FileRequirement{
		{Path: "cdk.json", Reason: "CDK project configuration"},
		{Path: "cdk.context.json", Reason: "CDK context values"},
	}
}

func (t *Tool) Diagnose(ctx context.Context, dir string, r tool.NodeReporter) error {
	return tool.DiagnoseDefaults(ctx, dir, t, tool.BinCheckerFrom(ctx), r)
}

func (t *Tool) Bootstrap(ctx context.Context, dir string, _ tool.NodeReporter) error {
	cfg := configFromCtx(ctx)
	opts, _ := tool.BootstrapOptionsFrom(ctx)

	profile := opts.Profile
	if profile == "" {
		profile = cfg.Profile
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return err
	}

	executionPolicies, permissionsBoundary, err := resolveBootstrapFlags(
		ctx, cfg, cctx, dir, profile, opts,
	)
	if err != nil {
		return err
	}

	templatePath, err := patchedBootstrapTemplate(ctx, dir)
	if err != nil {
		return err
	}
	defer os.Remove(templatePath)

	cdkArgs := cfg.cdkArgs(cctx.Qualifier)
	args := make([]string, 0, 1+len(cdkArgs)+8)
	args = append(args, "bootstrap")
	args = append(args, cdkArgs...)
	args = append(args, "--template", templatePath)
	// CdkArgs() may already include --profile from cfg.Profile. When the
	// bootstrap command receives a different --profile override (e.g. an admin
	// profile), strip the existing one to avoid passing --profile twice, which
	// causes CDK to use the wrong (first) profile.
	if profile != "" && profile != cfg.Profile {
		args = filterProfileArgs(args)
		args = append(args, "--profile", profile)
	}
	if executionPolicies != "" {
		args = append(args, "--cloudformation-execution-policies", executionPolicies)
	}
	if permissionsBoundary != "" {
		args = append(args, "--custom-permissions-boundary", permissionsBoundary)
	}
	return cmdexec.Run(ctx, dir, "cdk", args...)
}

func (t *Tool) Diff(ctx context.Context, dir string, _ tool.NodeReporter) error {
	cfg := configFromCtx(ctx)

	deployment, err := resolveDeployment(ctx, cfg, dir)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return err
	}

	cdkArgs := cfg.cdkArgs(cctx.Qualifier)
	args := make([]string, 0, 3+len(cdkArgs))
	args = append(args, "diff")
	args = append(args, cdkArgs...)
	args = append(args, cctx.Qualifier+"*Shared", cctx.Qualifier+"*"+deployment)
	return cmdexec.Run(ctx, dir, "cdk", args...)
}

func (t *Tool) Deploy(ctx context.Context, dir string, _ tool.NodeReporter) error {
	cfg := configFromCtx(ctx)
	opts, _ := tool.DeployOptionsFrom(ctx)

	deployment, err := resolveDeployment(ctx, cfg, dir)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return err
	}

	args := []string{"deploy", "--require-approval", "never"}
	if opts.Hotswap {
		args = append(args, "--hotswap")
	}
	args = append(args, cfg.cdkArgs(cctx.Qualifier)...)
	args = append(args, cctx.Qualifier+"*Shared", cctx.Qualifier+"*"+deployment)
	return cmdexec.Run(ctx, dir, "cdk", args...)
}

func (t *Tool) Inspections() []tool.Inspection {
	return []tool.Inspection{
		{Name: "endpoints", Description: "API Gateway endpoint URLs", Run: inspectOutputsByKey("GatewayURL")},
		{Name: "logs", Description: "CloudWatch log group names", Run: inspectOutputsByKey("LogGroup")},
		{Name: "1password-sync", Description: "1Password sync configuration", Run: inspect1PasswordSync},
	}
}

type stackInfo struct {
	Name   string
	Region string
}

func listDeploymentStacks(
	ctx context.Context, dir string, cfg *cdkConfig,
	cctx *cdkctx.CDKContext, deployment string,
) ([]stackInfo, error) {
	listArgs := append([]string{"list"}, cfg.cdkArgs(cctx.Qualifier)...)
	out, err := cmdexec.Output(ctx, dir, "cdk", listArgs...)
	if err != nil {
		return nil, err
	}

	var stacks []stackInfo
	for line := range strings.SplitSeq(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if !strings.HasSuffix(name, deployment) && !strings.HasSuffix(name, "Shared") {
			continue
		}
		region, ok := cctx.ResolveStackRegion(name)
		if !ok {
			continue
		}
		stacks = append(stacks, stackInfo{Name: name, Region: region})
	}
	return stacks, nil
}

func inspectOutputsByKey(outputKeySubstr string) func(context.Context, string, tool.NodeReporter) error {
	return func(ctx context.Context, dir string, r tool.NodeReporter) error {
		cfg := configFromCtx(ctx)

		deployment, err := resolveDeployment(ctx, cfg, dir)
		if err != nil {
			return err
		}

		cctx, err := cdkctx.Load(dir)
		if err != nil {
			return err
		}

		stacks, err := listDeploymentStacks(ctx, dir, cfg, cctx, deployment)
		if err != nil {
			return err
		}

		for _, stack := range stacks {
			if !strings.HasSuffix(stack.Name, deployment) {
				continue
			}
			outputs, err := cfnread.StackOutputs(ctx, stack.Region, cfg.Profile, stack.Name)
			if err != nil {
				r.Error(fmt.Sprintf("%s: (not deployed)", stack.Name))
				continue
			}

			var rows [][]string
			for k, v := range outputs {
				if strings.Contains(k, outputKeySubstr) {
					rows = append(rows, []string{k, v})
				}
			}
			if len(rows) > 0 {
				r.Section(fmt.Sprintf("%s (%s)", stack.Name, stack.Region))
				r.Table([]string{"OutputKey", "OutputValue"}, rows)
			}
		}
		return nil
	}
}

func inspect1PasswordSync(ctx context.Context, dir string, r tool.NodeReporter) error {
	cfg := configFromCtx(ctx)

	deployment, err := resolveDeployment(ctx, cfg, dir)
	if err != nil {
		return err
	}

	cctx, err := cdkctx.Load(dir)
	if err != nil {
		return err
	}

	stacks, err := listDeploymentStacks(ctx, dir, cfg, cctx, deployment)
	if err != nil {
		return err
	}

	var sharedStack, deploymentStack stackInfo
	for _, si := range stacks {
		if sharedStack.Name == "" && strings.HasSuffix(si.Name, "Shared") {
			sharedStack = si
		}
		if deploymentStack.Name == "" && strings.HasSuffix(si.Name, deployment) {
			deploymentStack = si
		}
	}

	if sharedStack.Name == "" {
		return errors.New("no shared stack found")
	}
	if deploymentStack.Name == "" {
		return errors.Newf("no deployment stack found for %s", deployment)
	}

	sharedOutputs, err := cfnread.StackOutputs(ctx, sharedStack.Region, cfg.Profile, sharedStack.Name)
	if err != nil {
		return err
	}

	deployOutputs, err := cfnread.StackOutputs(ctx, deploymentStack.Region, cfg.Profile, deploymentStack.Name)
	if err != nil {
		return err
	}

	samlARN := sharedOutputs["OnePasswordSAMLProviderARN"]
	roleARN := outputContaining(deployOutputs, "OnePasswordSyncRoleARN")
	secretName := outputContaining(deployOutputs, "OnePasswordSyncSecretName")

	r.Table([]string{"Setting", "Value"}, [][]string{
		{"SAML provider ARN", valueOrMissing(samlARN, "deploy shared stack first")},
		{"IAM role ARN", valueOrMissing(roleARN, "deploy deployment stack first")},
		{"Target region", sharedStack.Region},
		{"Target secret name", valueOrMissing(secretName, "deploy deployment stack first")},
	})

	return nil
}

func outputContaining(outputs map[string]string, substr string) string {
	for k, v := range outputs {
		if strings.Contains(k, substr) {
			return v
		}
	}
	return ""
}

func valueOrMissing(val, hint string) string {
	if val == "" {
		return "(not found - " + hint + ")"
	}
	return val
}

func ProfileFromConfig(cfg any) string {
	if c, ok := cfg.(cdkConfig); ok {
		return c.Profile
	}
	return ""
}

func configFromCtx(ctx context.Context) *cdkConfig {
	cfg := tool.ToolConfigFrom[cdkConfig](ctx)
	if cfg == nil {
		return &cdkConfig{}
	}
	return cfg
}

func resolveDeployment(ctx context.Context, cfg *cdkConfig, dir string) (string, error) {
	if d, ok := tool.DeploymentFrom(ctx); ok && d != "" {
		return d, nil
	}
	if cfg.DevStrategy == "iam-username" {
		return devstrategy.IAMDeployment(ctx, cfg.Profile)
	}
	claim, err := devslot.EnsureClaim(ctx, dir, cfg.Profile)
	if err != nil {
		return "", err
	}
	return claim.Slot, nil
}

func resolveBootstrapFlags(
	ctx context.Context, cfg *cdkConfig, cctx *cdkctx.CDKContext, dir, profile string, opts tool.BootstrapOptions,
) (executionPolicies, permissionsBoundary string, err error) {
	if cfg.PreBootstrap == nil {
		return opts.ExecutionPolicies, opts.PermissionsBoundary, nil
	}

	outputs, err := runPreBootstrap(ctx, cfg, cctx, dir, profile)
	if err != nil {
		return "", "", err
	}

	executionPolicies = opts.ExecutionPolicies
	if v := outputs["ExecutionPolicyArn"]; v != "" {
		if opts.ExecutionPolicies != "" {
			return "", "", errors.New(
				"--execution-policies cannot be used when pre-bootstrap stack provides ExecutionPolicyArn",
			)
		}
		executionPolicies = v
	}

	permissionsBoundary = opts.PermissionsBoundary
	if v := outputs["PermissionBoundaryName"]; v != "" {
		if opts.PermissionsBoundary != "" {
			return "", "", errors.New(
				"--permissions-boundary cannot be used when pre-bootstrap stack provides PermissionBoundaryName",
			)
		}
		permissionsBoundary = v
	}

	return executionPolicies, permissionsBoundary, nil
}

func runPreBootstrap(
	ctx context.Context, cfg *cdkConfig, cctx *cdkctx.CDKContext, dir, profile string,
) (map[string]string, error) {
	pb := cfg.PreBootstrap
	templatePath := filepath.Join(dir, pb.Template)

	if err := cfnvalidate.PreBootstrapTemplate(templatePath); err != nil {
		return nil, errors.Wrap(err, "validating pre-bootstrap template")
	}

	params, err := cfnparams.Resolve(pb.Parameters, cctx.ContextValues)
	if err != nil {
		return nil, errors.Wrap(err, "resolving pre-bootstrap parameters")
	}

	stackName := cctx.Qualifier + "-pre-bootstrap"

	fmt.Fprintf(os.Stderr, "Deploying pre-bootstrap stack %s...\n", stackName)
	if err := cfndeploy.Deploy(ctx, dir, profile, stackName, templatePath, params); err != nil {
		return nil, errors.Wrap(err, "deploying pre-bootstrap stack")
	}

	outputs, err := cfnread.StackOutputs(ctx, cctx.PrimaryRegion, profile, stackName)
	if err != nil {
		return nil, errors.Wrap(err, "reading pre-bootstrap stack outputs")
	}

	return outputs, nil
}

func filterProfileArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--profile" && i+1 < len(args) {
			i++
			continue
		}
		filtered = append(filtered, args[i])
	}
	return filtered
}

func patchedBootstrapTemplate(ctx context.Context, dir string) (string, error) {
	templateYAML, err := cmdexec.Output(ctx, dir, "cdk", "bootstrap", "--show-template")
	if err != nil {
		return "", errors.Wrap(err, "getting default bootstrap template")
	}

	patched, err := cfnpatch.AddDevSlotLifecycle([]byte(templateYAML), devSlotExpirationDays)
	if err != nil {
		return "", errors.Wrap(err, "patching bootstrap template")
	}

	tmpFile, err := os.CreateTemp("", "cdk-bootstrap-*.yaml")
	if err != nil {
		return "", errors.Wrap(err, "creating temp file for bootstrap template")
	}

	if _, err := tmpFile.Write(patched); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", errors.Wrap(err, "writing patched bootstrap template")
	}
	tmpFile.Close()

	return tmpFile.Name(), nil
}
