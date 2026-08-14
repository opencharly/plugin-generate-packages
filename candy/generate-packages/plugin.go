// Package generatepackages is the importable form of the charly COMMAND-class plugin
// contributing `charly generate-packages`, usable in BOTH placements (F8): COMPILED
// INTO charly in-process (charly imports this package + registers NewProvider()/
// NewMeta() via plugins_generated.go; `charly generate-packages <args>` dispatches
// IN-PROC via Invoke(OpRun) — dispatchInProcCommand) OR served OUT-OF-PROCESS (the
// cmd/serve shim; charly fork/execs the binary in CLI mode → CliMain). Both placements
// run the SAME effect (build the requested native packages via sdk/packagekit), so the
// command behaves identically regardless of placement — the placement-invisible
// command, the command-class analogue of candy/plugin-example-external. The dynamic
// Kong grammar (pass-through Args) is built host-side (externalCommandHolder) for both
// placements; only the dispatch transport differs.
//
// The plugin is the command surface ONLY: every nFPM/parsing/variant/optdepends
// function lives in sdk/packagekit (shared with the sdk's localpkg replacement). The
// plugin reads the `packaging:` section from the charly.yml named by --candy and builds
// the packages from that + the flags — no deps table, no other metadata resource.
package generatepackages

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
)

const calver = "2026.226.0001"

// NewProvider returns the command provider for in-proc registration (compiled-in) or out-of-proc serving.
func NewProvider() pb.ProviderServer { return &provider{} }

// NewMeta advertises command:generate-packages via sdk.NewMeta → BuildCapabilities so
// the COMPILED-IN path registers it as a command provider (buildUnitInProc →
// inprocProvider Class=command; the host builds its dynamic Kong grammar + dispatches
// Invoke(OpRun)). The served schema carries no #*Input def — a command's args are
// pass-through CLI tokens, not a structured plugin_input — so the capability has no
// InputDef.
func NewMeta() pb.PluginMetaServer {
	return sdk.NewMeta(calver,
		[]sdk.ProvidedCapability{{Class: "command", Word: "generate-packages"}},
		nil)
}

// CliMain is the OUT-OF-PROCESS CLI-mode entry (charly fork/execs the binary with the
// pass-through tokens after `charly generate-packages`). It runs the SAME effect as the
// in-proc Invoke(OpRun) path.
func CliMain(args []string) int {
	if err := runFromArgs(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type provider struct{ pb.UnimplementedProviderServer }

// Invoke handles OpRun for the COMPILED-IN (in-proc) dispatch: decode the pass-through
// {args} and run the command effect in charly's own process. (Out-of-process dispatch
// is fork/exec → CliMain, never this gRPC path.)
func (provider) Invoke(_ context.Context, req *pb.InvokeRequest) (*pb.InvokeReply, error) {
	if req.GetOp() != sdk.OpRun {
		return nil, fmt.Errorf("generate-packages: unsupported op %q (only %q)", req.GetOp(), sdk.OpRun)
	}
	var in struct {
		Args []string `json:"args"`
	}
	if len(req.GetParamsJson()) > 0 {
		if err := json.Unmarshal(req.GetParamsJson(), &in); err != nil {
			return nil, fmt.Errorf("generate-packages: decode args: %w", err)
		}
	}
	if err := runFromArgs(in.Args); err != nil {
		return nil, err
	}
	return &pb.InvokeReply{}, nil
}
