# nix-auth

Tired of hitting rate limits when running `nix flake update`? Trying to
fetch a private repository in your flake inputs or builtin fetchers?

Nix supports setting access-tokens in your Nix config. This tool makes it easy
to get those tokens in the right place.

## Features

- OAuth device flow authentication when possible (no manual token creation needed)
- Support for multiple providers (GitHub, GitHub Enterprise, GitLab, Gitea, and Forgejo)
- Token storage in `~/.config/nix/nix.conf`
- Token validation and status checking
- Automatic backup creation before modifying configuration

## Installation

### Using Nix Flakes

Run directly without installation:

```bash
nix run github:numtide/nix-auth -- login
```

Install into your profile:

```bash
nix profile install github:numtide/nix-auth
```

Or add to your system configuration:

```nix
{
  inputs.nix-auth.url = "github:numtide/nix-auth";

  # In your system packages
  environment.systemPackages = [
    inputs.nix-auth.packages.${system}.default
  ];
}
```

### Using Go

```bash
go install github.com/numtide/nix-auth@latest
```

### From Source

```bash
git clone https://github.com/numtide/nix-auth
cd nix-auth
go build .
```

## Usage

### Login

Authenticate with a provider:

```bash
# Using provider aliases
nix-auth login                        # defaults to github
nix-auth login github
nix-auth login gitlab
nix-auth login gitea
nix-auth login codeberg

# Using hosts with auto-detection
nix-auth login github.com
nix-auth login gitlab.company.com     # auto-detects provider type
nix-auth login gitea.company.com      # auto-detects provider type

# Explicit provider specification
nix-auth login git.company.com --provider forgejo
nix-auth login github.company.com --provider github --client-id <your-client-id>
nix-auth login gitlab.company.com --provider gitlab --client-id <your-application-id>
```

The tool will:
1. Display a one-time code
2. Open your browser to the provider's device authorization page
3. Wait for you to authorize the application
4. Save the token to `~/.config/nix/nix.conf`

**Note for self-hosted instances**:
- **GitHub Enterprise**: You'll need to create an OAuth App and provide the client ID via `--client-id`
- **GitLab self-hosted**: You'll need to create an OAuth application and provide the client ID via `--client-id`
- **Gitea/Forgejo**: Uses Personal Access Token flow instead of OAuth device flow (these platforms don't support device flow yet)

The tool will guide you through this process if the client ID is not provided. You can also set the `GITHUB_CLIENT_ID` or `GITLAB_CLIENT_ID` environment variables as an alternative to the `--client-id` flag.

### Check Status

View all configured tokens:

```bash
nix-auth status
```

### Logout

Remove a token interactively:

```bash
nix-auth logout
```

Remove a specific provider's token:

```bash
nix-auth logout github
```

Remove a token for a specific host:

```bash
nix-auth logout --host github.company.com
```

## How It Works

The tool manages the `access-tokens` configuration in your `~/.config/nix/nix.conf` file. This allows Nix to authenticate when fetching flake inputs from private repositories or builtins fetchers, and hitting rate limits.

Example configuration added by this tool:
```
access-tokens = github.com=ghp_xxxxxxxxxxxxxxxxxxxx gitlab.com=glpat-xxxxxxxxxxxx github.company.com=ghp_yyyyyyyy
```

## Security

- Tokens are stored locally in your Nix configuration
- The tool creates automatic backups before modifying your configuration
- Uses OAuth device flow for secure authentication
- Minimal required permissions (only necessary scopes for accessing repositories)

## Future Plans

- Support for more providers (Bitbucket, etc.)
- Token expiration notifications
- Integration with system keychains for secure storage (will require patching
    Nix)

## License

MIT
