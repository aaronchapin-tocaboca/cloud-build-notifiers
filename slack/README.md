# Cloud Build Slack Notifier

This notifier uses [Slack Webhooks](https://api.slack.com/messaging/webhooks) to
send notifications to your Slack workspace.

This notifier runs as a container via Google Cloud Run and responds to
events that Cloud Build publishes via its
[Pub/Sub topic](https://cloud.google.com/cloud-build/docs/send-build-notifications).

For detailed instructions on setting up this notifier,
see [Configuring Slack notifications](https://cloud.google.com/cloud-build/docs/configuring-notifications/configure-slack).

## Configuration Variables

This notifier expects the following fields in the `delivery` map to be set:

- `webhook_url`: The `secretRef: <Slack-webhook-URL>` map that references the
Slack webhook URL resource path in the `secrets` section.

## Building and pushing this standalone

1. From the root directory: 

`sudo docker build . -f=./slack/Dockerfile --tag=slack-notifier`

2. Make sure you're authed with google cloud to the `toca-days-ops` project

3. Tag your local image for pushing

`docker tag slack-notifier us-docker.pkg.dev/toca-days-ops/toca-days/slack-notifier`

4. Push the image to artifact registry

`docker push us-docker.pkg.dev/toca-days-ops/toca-days/slack-notifier`

5. Update the `slack-build-notification` [Cloud Run service on Google Cloud](https://console.cloud.google.com/run?project=toca-days-ops). Configure it to use the newly-pushed image, and carry on!