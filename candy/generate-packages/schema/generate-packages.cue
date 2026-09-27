// plugin-generate-packages's OWN self-contained CUE schema — the SINGLE SOURCE for
// this plugin's declaration surface, served over Describe exactly like every other
// plugin's schema (there is no schema-less plugin):
//
//  1. SERVE over Describe — the host splices `base ++ plugin` at the load gate
//     (registerPluginUnitSchema), so the plugin's declarations travel WITH it and
//     a self-contained schema that will not splice is a LOUD load failure.
//  2. DOCUMENT — `charly docs generate` renders this plugin's page from its
//     providers + this schema + the candy `description:`.
//
// command:generate-packages's authored input is its pass-through CLI grammar (the
// `--candy` + build flags), the `{args: [...]}` envelope the host dispatches to
// Invoke(OpRun) — not a structured plugin_input — so this schema DOCUMENTS the
// command contract (no #*Input def). SELF-CONTAINED: it references no base def, so
// it compiles STANDALONE (the property that lets the SDK compile it serve-side).
#GeneratePackagesPlugin: {
	// The capability word the plugin serves.
	command: "generate-packages"

	// What the command does, in one line (the public-docs surface): build the
	// requested native packages from a candy's `packaging:` section.
	contract: string & !=""
}
