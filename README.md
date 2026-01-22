# tfsnap

A CLI tool for managing Terraform configurations during provider development.

## Installation

```bash
go install github.com/phergul/tfsnap@latest
```

## Demo

### 1. Initialize tfsnap

Initialize tfsnap in your working directory:

```bash
tfsnap init
```

This will prompt you for:
- Provider directory path (path to your local provider source code)

Optionally configure a custom local source for provider development.

Alternatively, load from a config file:

```bash
tfsnap init --config path/to/config.yaml
```

### 2. Inject Resources

Inject example resource configurations into your `main.tf`:

```bash
# Inject a single resource
tfsnap inject s3_bucket

# Inject multiple resources
tfsnap inject s3_bucket ec2_instance

# Inject with a specific version
tfsnap inject s3_bucket --version 5.0.0

# Inject a skeleton instead of an example
tfsnap inject s3_bucket --skeleton

# Include dependent resources
tfsnap inject s3_bucket --dependencies
```

### 3. Create Snapshots

Save your current Terraform configuration as a snapshot:

```bash
# Basic snapshot
tfsnap snapshot save my-snapshot

# Snapshot with description
tfsnap snapshot save my-snapshot --description "Testing S3 bucket configuration"

# Include provider binary
tfsnap snapshot save my-snapshot --include-binary

# Include git information
tfsnap snapshot save my-snapshot --include-git
```

### 4. Manage Snapshots

Browse, load, and delete snapshots using the interactive TUI:

```bash
# Open snapshot management interface
tfsnap snapshot
```

The TUI provides:
- Browse all snapshots with detailed information displayed in the right pane
- Press `Enter` to load a snapshot (creates autosave before loading)
- Press `d` to delete a snapshot
- Navigate with arrow keys or `j`/`k`
- Press `q` or `Esc` to quit

### 5. Manage Templates

Save and reuse resource configurations as templates using the interactive TUI:

```bash
# Save a resource from main.tf as a template
tfsnap template save my-template

# Browse, inject, or delete templates
tfsnap template
```

The template TUI provides:
- Browse all saved templates with resource content preview
- Press `Enter` to inject a template into main.tf
- Press `d` to delete a template
- Navigate with arrow keys or `j`/`k`
- Press `q` or `Esc` to quit

### 6. Additional Commands

```bash
# Restore autosaved snapshot
tfsnap restore

# Clean up terraform files
tfsnap clean

# Clean up excluding specific files
tfsnap clean --exclude .terraform.lock.hcl
```

## Commands

### `tfsnap init`

Initialize tfsnap in the current directory. Automatically detects provider information from the provider directory.

**Flags:**
- `-c, --config <file>`: Load configuration from a YAML file

### `tfsnap inject <resources...>`

Inject Terraform resource examples into your configuration.

**Flags:**
- `-v, --version <version>`: Specify provider version for the resource
- `-s, --skeleton`: Generate a skeleton instead of an example
- `-l, --local`: Use local provider binary
- `-d, --dependencies`: Include dependent resources

### `tfsnap snapshot`

Open the interactive snapshot management interface. Browse snapshots with detailed information including creation time, provider details, git information, and resource counts. Load or delete snapshots directly from the TUI.

**Actions:**
- `Enter`: Load the selected snapshot
- `d`: Delete the selected snapshot
- `↑/↓` or `j/k`: Navigate between snapshots
- `q` or `Esc`: Quit

### `tfsnap snapshot save <name>`

Save the current Terraform configuration as a snapshot.

**Flags:**
- `-d, --description <text>`: Add a description to the snapshot
- `-b, --include-binary`: Include the provider binary
- `-g, --include-git`: Include git branch and commit information
- `-p, --persist`: Persist the saved configuration instead of clearing it

### `tfsnap template`

Open the interactive template management interface. Browse saved resource templates and inject them into main.tf or delete them.

**Actions:**
- `Enter`: Inject the selected template into main.tf
- `d`: Delete the selected template
- `↑/↓` or `j/k`: Navigate between templates
- `q` or `Esc`: Quit

### `tfsnap template save <name>`

Save a resource from main.tf as a reusable template. Opens a TUI to select which resource to save.

### `tfsnap restore`

Restore the automatically saved snapshot. tfsnap creates autosaves before operations that modify your configuration.

### `tfsnap clean`

Clean up local Terraform files including .terraform.lock.hcl, terraform.tfstate, and terraform.tfstate.backup.

**Flags:**
- `-e, --exclude <file>`: Files to exclude from cleanup (can be specified multiple times)

### `tfsnap version <version>`

Change the provider version for the current configuration.

**Flags:**
- `-l, --local`: Use local provider version

## Configuration

tfsnap stores its configuration in `.tfsnap/config.yaml` in your working directory:

```yaml
config_path: /path/to/project/.tfsnap/config.yaml
working_directory: /path/to/project
snapshot_directory: /path/to/project/.tfsnap/snapshots
provider:
  name: aws
  provider_directory: /path/to/terraform-provider-aws
  source_mappings:
    local_source: local/aws
    registry_source: hashicorp/aws
```