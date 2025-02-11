# Ladon

A dead-simple index page for your homeserver, protected via OpenID Connect.

## Deployment

### Configuration

#### KDL config for links

Ladon relies on a single KDL file for displaying your links. Here's an example:

```kdl
group "Media" {
  link "Plex" url="https://plex.mydomain.com"
  link "Calibre CWA" url="https://calibre.mydomain.com"
  link "Neko" url="https://<Tailscale IP>:8000"
}

group "Collaboration" {
  link "Jira" url="https://jira.myworkdomain.com"
  link "Confluence" url="https://docs.myworkdomain.com"
}
```

The container image expects this file to be at `/data/links.kdl`, so you'll
need to perform a volume binding at deploy time to ensure that file is
available.

#### Environment Variables

After your config file, you'll need to set some environment variables for
OpenID Connect. Bog-standard OAuth2 is not currently supported, since Ladon was
mainly built with Pocket ID in mind.

**All value are required.**

| Variable Name | Description |
| `SESSION_SECRET` | 16 character string used to encrypt and sign session cookies. Make sure this is a randomly-generated value. |
| `OIDC_CLIENT_ID` | OAuth2 client ID from your OIDC provider. |
| `OIDC_CLIENT_SECRET` | OAuth2 client secret from your OIDC provider. |
| `OIDC_ISSUER` | Issuer of your OIDC provider. In the case of Pocket ID, this will be simply your root domain with protocol. For other OpenID providers, this should be the start of your discovery URL, i.e. your domain minus the `/.well-known/openid-configuriation` at the end. |
| `LADON_DOMAIN` | Domain where you are hosting Ladon. This is required for informing your OpenID provider of the correct callback URL. |

### Starting Ladon

With the config out of the way, you can deploy Ladon with your container
runtime of choice.

#### Docker Compose

```yaml
# docker-compose.yml

services:
  ladon:
    image: ghcr.io/puregarlic/ladon
    environment:
      SESSION_SECRET: changemechangeme
      OIDC_CLIENT_ID: changeme
      OIDC_CLIENT_SECRET: qwfpjljujehnneharst
      OIDC_ISSUER: https://oid.mydomain.com
      LADON_DOMAIN: https://mydomain.com

    # If you'd rather use a dotenv file, comment the above and uncomment below:
    # env_file: ".env"
      
    ports:
      - "4000:4000"
    volumes:
      # In your data directory, make sure you've made `links.kdl`
      - ./data:/data
```

#### Quadlet

The below example assumes you want Ladon to start at boot, and that your
environment variables are stored in a `.env` file located at `/my/environment/.env`.

```ini
[Unit]
Description="Links index"

[Container]
AutoUpdate=registry
Image=ghcr.io/puregarlic/ladon
PublishPort=4000:4000tcp

# Update these values for your deployment
Volume=/my/data/path:/data
EnvironmentFile=/my/environment/.env

[Install]
WantedBy=default.target
```

## Potentially-Asked Questions

> Can I theme my page?

Not at this time, but potentially in the future. The current theme is Rose Pine,
if you want to look it up.

> Can I add extra data to my links, e.g. descriptions?

Not at this time again, but maybe in the future. If you really want, you can
nest groups, but there's really no significant difference in the formatting as
a result of doing such.

> Can I show certain links to certain users?

Nope, but maybe--you guessed it--in the future. Contributions are welcome!

> Why Ladon?

According to [Wikipedia](https://en.wikipedia.org/wiki/Ladon_(mythology)),
_Ladon was the serpent-like dragon that twined and twisted around the tree in
the Garden of the Hesperides and guarded the golden apples._ The apps on
your homeserver are kind of like golden apples (for hackers), so maybe this
program can be the serpent-like dragon to guard them for you.

At least on the surface, anyway. It's worth noting that Ladon is not a
replacement for safely and securely configuring your applications. Ladon was
only designed to make the lives of your friends and family easier without
broadcasting an itemized list of potential vulnerabilities.
