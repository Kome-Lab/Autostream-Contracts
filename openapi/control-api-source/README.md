# Control API authoring location

The ordered raw fragments now live in `authoring/control-api/`, with the schema
fragments in `authoring/discord-start-job/`. The manifest for both public
artifacts is `authoring/artifact-layout.json`.

Run `python scripts/assemble-control-api.py` to regenerate the public artifacts,
or add `--check` to verify their raw bytes. Existing public paths and OpenAPI
reference resolution are unchanged. See `authoring/README.md` for the immutable
Git-object verification command and source-size policy.
