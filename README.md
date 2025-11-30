# tfsnap - WIP

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
- Provider name (e.g., `aws`, `google`, `azurerm`)
- Provider directory path (path to your local provider source)
- Local source mapping (e.g., `local.com/provider/name`)
- Registry source mapping (e.g., `hashicorp/aws`)

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

```bash
# List all snapshots
tfsnap snapshot list

# List with detailed resource information
tfsnap snapshot list --detailed

# Load a snapshot
tfsnap snapshot load my-snapshot

# Delete a snapshot
tfsnap snapshot delete my-snapshot
```

## Commands

### `tfsnap init`

Initialize tfsnap in the current directory.

**Flags:**
- `-c, --config <file>`: Load configuration from a YAML file

### `tfsnap inject <resources...>`

Inject Terraform resource examples into your configuration.

**Flags:**
- `-v, --version <version>`: Specify provider version for the resource
- `-s, --skeleton`: Generate a skeleton instead of an example
- `-l, --local`: Use local provider binary
- `-d, --dependencies`: Include dependent resources

### `tfsnap snapshot save <name>`

Save the current Terraform configuration as a snapshot.

**Flags:**
- `-d, --description <text>`: Add a description to the snapshot
- `-b, --include-binary`: Include the provider binary
- `-g, --include-git`: Include git branch and commit information

### `tfsnap snapshot load <name>`

Restore a previously saved snapshot.

### `tfsnap snapshot list`

List all saved snapshots.

**Flags:**
- `-d, --detailed`: Show detailed resource information

### `tfsnap snapshot delete <name>`

Delete a snapshot.

### `tfsnap version <version>`

Change the provider version for the current snapshot.

**Flags:**
- `-l`, `--local`: Use local provider version

### `tfsnap completion`

Generate shell completion scripts for bash or zsh.

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