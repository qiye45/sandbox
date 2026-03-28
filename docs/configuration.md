# Sandbox Configuration

Sandbox provides two layers of configuration: a **Global Configuration** that controls how the CLI and Docker containers behave generally, and a **Workspace Configuration** that controls environment setup on a per-project basis.

## Global Configuration (`~/.sandbox/config.yaml`)

When you first run `sandbox`, a default configuration file is automatically generated at `~/.sandbox/config.yaml`. This file dictates the security bounds, default images, and paths used by the application.

### `images`
A mapping of agent/runner names to their default Docker base images. You can override these to point to your own custom images or specific tags.
- The `default` key is used when an agent cannot be inferred from the command.

### `env_whitelist`
A list of environment variable names (globs are supported) that are permitted to be forwarded from your host computer into the container.
- Example: `["LANG", "SHELL", "TERM"]`

### `env_blocklist`
A rigid list of environment variable names (globs are supported) that will **never** be forwarded to the container, even if matched by the whitelist. This is used to protect sensitive keys.
- Example: `["AWS_*", "GITHUB_TOKEN", "OPENAI_API_KEY"]`

### `container`
Controls basic Docker container execution semantics.
- `timeout` (string): Maximum execution time before the container is forcibly stopped (e.g., `30m`, `1h`).
- `network_mode` (string): The Docker network mode to use (default: `bridge`).
- `remove` (boolean): Whether to automatically remove the container after it stops executing (default: `true`).

### `security`
Dictates the resource limitations and sandboxing primitives.
- `memory_limit` (string): Maximum memory a sandbox can consume (e.g., `4GB`, `512MB`).
- `cpu_quota` (int64): CPU quota in microseconds per 100ms period (0 for unlimited).
- `pids_limit` (int64): The maximum number of processes allowed inside the sandbox (default: `512`).
- `seccomp_profile_path` (string): Optional path to a custom seccomp JSON profile. If left blank, Sandbox uses its default strict profile.
- `read_only_root` (boolean): Mounts the container rootfs as read-only.
- `user_mapping` (string): The `uid:gid` the container processes should run as (default: `65534:65534`).
- `drop_capabilities` (string array): Additional Linux capabilities to drop.

### `logging`
Logging level preferences.
- `level` (string): e.g., `debug`, `info`, `warn`.
- `format` (string): `console` or `json`.

### `paths`
Internal mounting rules for the sandbox.
- `workspace` (string): The mount path inside the container where your code resides (default: `/work`).
- `config_dir` (string): The directory on your host containing sandbox configurations (default: `~/.sandbox`).
- `cache_dir` (string): The directory on your host mapped to `~/.cache` inside the container to speed up subsequent agent runs for package managers like `npm`, `pip`, or `bun`.

---

## Workspace Configuration (`.sandbox.yml`)

You can define a `.sandbox.yml` file at the root of your project directory to dictate how the sandbox should bootstrap itself prior to running your agent command.

### `setup`
A list of shell commands to execute inside the container *before* the main agent or runtime command executes.
This allows you to pre-install dependencies required by an agent dynamically without needing to configure a custom Dockerfile.

**Example `.sandbox.yml`:**
```yaml
setup:
  - apt-get update && apt-get install -y jq
  - pip install -r requirements.txt
```
When `sandbox run python main.py` is invoked in this directory, the sandbox will construct a temporary entrypoint, execute the `apt-get` and `pip` steps, and then seamlessly pass control to `python main.py`.
