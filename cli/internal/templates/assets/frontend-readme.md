# CloudCC Frontend Assets

This directory is managed by the Go skill template without creating an external frontend build project.

- Put pagecomponent source files under `frontend/pagecomponents/<name>/`.
- Put prebuilt UMD bundles under `frontend/build/`.
- `cloudcc publish pagecomponent <name>` uploads an existing bundle; it does not run a frontend build.
