![Magout](./magout_logo.svg)

Magout is a Kubernetes operator for Mastodon, which allows you to deploy Mastodon servers really easily on Kubernetes. Magout is being used for [mstdn.anqou.net](https://mstdn.anqou.net/).

Currently, Magout supports Kubernetes 1.29, 1.30, and 1.31.

## Quick Start

Install Magout's cluster-wide resources such as CRDs via Helm.

```
helm install \
    --repo https://ushitora-anqou.github.io/magout \
    magout-cluster-wide \
    magout-cluster-wide
```

Install a Mastodon server managed by the Magout controller. First, write a values.yaml:

```yaml
serverName: mastodon.test # Change here to your domain.

mastodonVersion:
  image: ghcr.io/mastodon/mastodon:v4.2.12 # Change here to the latest version.
  streamingImage: ghcr.io/mastodon/mastodon:v4.2.12 # Change here to the latest version.

mastodonServer:
  web:
    envFrom:
      - secretRef:
          # Change here to your Secret name where Mastodon's environment variables
          # such as LOCAL_DOMAIN, DB_HOST, and REDIS_HOST are defined.
          name: secret-env-web
  sidekiq:
    envFrom:
      - secretRef:
          # Change here.
          name: secret-env-sidekiq
  streaming:
    envFrom:
      - secretRef:
          # Change here.
          name: secret-env-streaming
```

Then, use helm:

```
helm install \
    --create-namespace \
    --namespace mastodon \
    --repo https://ushitora-anqou.github.io/magout \
    -f values.yaml \
    magout \
    magout
```

Magout will then start the necessary migration jobs and deployments for nginx, web, streaming, and sidekiq.

When upgrading Mastodon, all you need to do is edit the `image` and `streamingImage` field in the values.yaml file and run `helm upgrade`.
Magout will take care of the necessary DB migrations and roll out the new deployments.

## Features & Tips

- You can restart web, streaming, and/or sidekiq pods on a regular schedule by using the `periodicRestart` field. [See this test manifest](https://github.com/ushitora-anqou/magout/blob/fa34514da5ee78d01562f153a7f54a2883d59128/test/e2e/testdata/values-v4.3.0b1-restart.yaml#L40-L43) for details.
- You can add your favourite annotations to the pods. [See this test manifest](https://github.com/ushitora-anqou/magout/blob/master/test/e2e/testdata/values-v4.3.0b1-restart.yaml#L31-L32). This feature is especially useful in combination with [stakater/Reloader](https://github.com/stakater/Reloader) to reload configmaps and secrets specified in the MastodonServer resources.

## License

- `charts/magout-mastodon-server/templates/gateway.yaml`
  - A fork of Mastodon's `dist/nginx.conf`. See the file for the details.

The rest of the files are licensed under AGPL-3.0:

```
Magout -- A Kubernetes Operator for Mastodon
Copyright (C) 2024 Ryotaro Banno

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see http://www.gnu.org/licenses/.
```
