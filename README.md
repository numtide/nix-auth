# nix-auth

Tired of hitting rate limits when running `nix flake update`? Trying to
fetch a private repository in your flake inputs or builtin fetchers?

Nix supports setting access-tokens in your Nix config. This tool makes it easy
to get those tokens in the right place.

## Features

- OAuth device flow authentication (no manual token creation needed)
- Support for multiple providers (GitHub and GitLab)
- Secure token storage in `~/.config/nix/nix.conf`
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

Authenticate with GitHub (default provider):

```bash
nix-auth login
```

Authenticate with GitLab:

```bash
nix-auth login gitlab
```

The tool will:
1. Display a one-time code
2. Open your browser to the provider's device authorization page
3. Wait for you to authorize the application
4. Save the token to `~/.config/nix/nix.conf`

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

Or remove a specific provider's token:

```bash
nix-auth logout github
```

## How It Works

The tool manages the `access-tokens` configuration in your `~/.config/nix/nix.conf` file. This allows Nix to authenticate when fetching flake inputs from private repositories or builtins fetchers, and hitting rate limits.

Example configuration added by this tool:
```
access-tokens = github.com=ghp_xxxxxxxxxxxxxxxxxxxx gitlab.com=glpat-xxxxxxxxxxxx
```

## Security

- Tokens are stored locally in your Nix configuration
- The tool creates automatic backups before modifying your configuration
- Uses OAuth device flow for secure authentication
- Minimal required permissions (only necessary scopes for accessing repositories)

## Future Plans

- Support for more providers; Gitea / Forgego / GitHub Enterprise / ...
- Token expiration notifications
- Integration with system keychains for secure storage (will require patching
    Nix)

## License

MIT
