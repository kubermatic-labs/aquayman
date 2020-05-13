# Aquayman

Aquayman (**A Quay Man**ager) allows you to declare your teams, robots and repository permissions
in a declarative way (by virtue of a YAML file), which will then be applies to your Quay.io
organization. This enables consistent, always up-to-date team members and access rules.

## Features

* Manages organization teams, robots and repository permissions.
* Exports the current state as a starter config file.
* Previews any action taken, for greater peace of mind.

## Installation

We strongly recommend that you use an [official release][3] of Aquayman.

_The code and sample YAML files in the master branch of are under active development and are not guaranteed to be stable. Use them at your own risk!_

## Build From Source

This project uses Go 1.14 and Go modules for its dependencies. You can get it via `go get`:

```bash
GO111MODULE=on go get github.com/kubermatic-labs/aquayman
```

## Mode Of Operation

Whenever Aquayman synchronizes an organization, it will perform these steps:

1. Ensure only the robots defined in the configuration file exist and that their
   description is up-to-date.
2. Ensure only the teams defined in the configuration file exist. For each team,
   adjust (add or remove) the members.
3. List all existing repositories and for each

   1. Find a matching repository configuration, based on the name. This can be
      either an exact match, or a glob expression match.
   2. If no configuration is found, ignore the repository.
   3. Otherwise, adjust the assigned teams and individual users/robots.

## Usage

You need an OAuth2 token to authenticate against the API. In your organization settings
on Quay.io, you can create an application and for it you can then generate a token. Export
it as the environment variable `AQUAYMAN_TOKEN`:

```bash
export AQUAYMAN_TOKEN=thisisnotarealtokenbutjustanexample
```

### Configuration

Except for the OAuth2 token, all configuration happens in a YAML file. See the annotated
`config.example.yaml` for more information or let Aquayman generate a config for you by
exporting your current settings. See the next section for more information on this.

### Validating

It's possible to only validate a configuration file for syntactic correctness by running
Aquayman with the `-validate` flag:

```bash
aquayman -config myconfig.yaml -validate
2020/04/16 23:14:20 ✓ Configuration is valid.
```

Aquayman exits with code 0 if the config is valid, otherwise with a non-zero code.

### Exporting

To get started, Aquayman can export your existing Quay.io settings into a configuration file.
For this to work, prepare a fresh configuration file and put your organisation name in it.
You can skip everything else:

```yaml
organization: exampleorg
```

Now run Aquayman with the `-export` flag:

```bash
aquayman -config myconfig.yaml -export
2020/04/16 23:14:38 ► Exporting organization exampleorg
2020/04/16 23:14:38 ⇄ Exporting robots…
2020/04/16 23:14:39   ⚛ mybot
2020/04/16 23:14:39 ⇄ Exporting repositories…
2020/04/16 23:14:40   ⚒ myapp
2020/04/16 23:14:42 ⇄ Exporting teams…
2020/04/16 23:14:42   ⚑ owners
2020/04/16 23:14:43 ✓ Export successful.
```

Depending on your teams and repositories this can take a few minutes to run. Afterwards the
`myconfig.yaml` will have been updated to contain an exact representation of your settings:

```yaml
organisation: exampleorg
teams:
  - name: owners
    role: admin
    members:
      - exampleorg+mybot
repositories:
  - name: myapp
    teams:
      owners: admin
robots:
  - name: mybot
    description: Just an example bot.
```

### Synchronizing

Synchronizing means updating Quay.io to match the given configuration file. It's as simple
as running Aquayman:

```bash
aquayman -config myconfig.yaml
2020/04/16 23:32:00 ► Updating organization exampleorg…
2020/04/16 23:32:00 ⇄ Syncing robots…
2020/04/16 23:32:00   ✎ ⚛ mybot
2020/04/16 23:32:01   - ⚛ thisbotshouldnotexist
2020/04/16 23:32:01 ⇄ Syncing teams…
2020/04/16 23:32:01   ✎ ⚑ owners
2020/04/16 23:32:01     + ♟ exampleorg+mybot
2020/04/16 23:32:01 ⇄ Syncing repositories…
2020/04/16 23:32:02 ⚠ Run again with -confirm to apply the changes above.
```

Aquayman by default only shows a preview of things it would do. Run it with `-confirm` to let
the magic happen.

```bash
aquayman -config myconfig.yaml -confirm
2020/04/16 23:32:10 ► Updating organization exampleorg…
2020/04/16 23:32:10 ⇄ Syncing robots…
2020/04/16 23:32:10   ✎ ⚛ mybot
2020/04/16 23:32:11   - ⚛ thisbotshouldnotexist
2020/04/16 23:32:11 ⇄ Syncing teams…
2020/04/16 23:32:11   ✎ ⚑ owners
2020/04/16 23:32:11     + ♟ exampleorg+mybot
2020/04/16 23:32:11 ⇄ Syncing repositories…
2020/04/16 23:32:12 ✓ Permissions successfully synchronized.
```

## Troubleshooting

If you encounter issues [file an issue][1] or talk to us on the [#kubermatic-labs channel][12] on the [Kubermatic Slack][15].

## Contributing

Thanks for taking the time to join our community and start contributing!

Feedback and discussion are available on [the mailing list][11].

### Before you start

* Please familiarize yourself with the [Code of Conduct][4] before contributing.
* See [CONTRIBUTING.md][2] for instructions on the developer certificate of origin that we require.
* Read how [we're using ZenHub][13] for project and roadmap planning

### Pull requests

* We welcome pull requests. Feel free to dig through the [issues][1] and jump in.

## Changelog

See [the list of releases][3] to find out about feature changes.

[1]: https://github.com/kubermatic-labs/aquayman/issues
[2]: https://github.com/kubermatic-labs/aquayman/blob/master/CONTRIBUTING.md
[3]: https://github.com/kubermatic-labs/aquayman/releases
[4]: https://github.com/kubermatic-labs/aquayman/blob/master/CODE_OF_CONDUCT.md

[11]: https://groups.google.com/forum/#!forum/kubermatic-dev
[12]: https://kubermatic.slack.com/messages/kubermatic-labs
[13]: https://github.com/kubermatic-labs/aquayman/blob/master/Zenhub.md
[15]: http://slack.kubermatic.io/
