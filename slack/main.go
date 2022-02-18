// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-build-notifiers/lib/notifiers"
	log "github.com/golang/glog"
	"github.com/slack-go/slack"
	cbpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

const (
	webhookURLSecretName = "webhookUrl"
)

func main() {
	if err := notifiers.Main(new(slackNotifier)); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

type slackNotifier struct {
	filter notifiers.EventFilter

	webhookURL string
}

func (s *slackNotifier) SetUp(ctx context.Context, cfg *notifiers.Config, sg notifiers.SecretGetter, _ notifiers.BindingResolver) error {
	prd, err := notifiers.MakeCELPredicate(cfg.Spec.Notification.Filter)
	if err != nil {
		return fmt.Errorf("failed to make a CEL predicate: %w", err)
	}
	s.filter = prd

	wuRef, err := notifiers.GetSecretRef(cfg.Spec.Notification.Delivery, webhookURLSecretName)
	if err != nil {
		return fmt.Errorf("failed to get Secret ref from delivery config (%v) field %q: %w", cfg.Spec.Notification.Delivery, webhookURLSecretName, err)
	}
	wuResource, err := notifiers.FindSecretResourceName(cfg.Spec.Secrets, wuRef)
	if err != nil {
		return fmt.Errorf("failed to find Secret for ref %q: %w", wuRef, err)
	}
	wu, err := sg.GetSecret(ctx, wuResource)
	if err != nil {
		return fmt.Errorf("failed to get token secret: %w", err)
	}
	s.webhookURL = wu

	return nil
}

func (s *slackNotifier) SendNotification(ctx context.Context, build *cbpb.Build) error {
	if !s.filter.Apply(ctx, build) {
		return nil
	}

	log.Infof("sending Slack webhook for Build %q (status: %q)", build.Id, build.Status)
	msg, err := s.writeMessage(build)
	if err != nil {
		return fmt.Errorf("failed to write Slack message: %w", err)
	}

	return slack.PostWebhook(s.webhookURL, msg)
}

func (s *slackNotifier) writeMessage(build *cbpb.Build) (*slack.WebhookMessage, error) {
	branchName := build.Substitutions["BRANCH_NAME"]
	commitSha := build.Substitutions["COMMIT_SHA"]
	triggerName := build.Substitutions["TRIGGER_NAME"]

	var txt string
	var clr string
	switch build.Status {
	case cbpb.Build_SUCCESS:
		clr = "good"
		txt = fmt.Sprintf(
			"A new build of %s has succeeded! :tocarocket:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	case cbpb.Build_FAILURE:
		clr = "danger"
		txt = fmt.Sprintf(
			"A new build of %s has failed! :tocascream:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	case cbpb.Build_INTERNAL_ERROR:
		clr = "danger"
		txt = fmt.Sprintf(
			"A new build of %s has had an internal error! :tocascream:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	case cbpb.Build_TIMEOUT:
		clr = "danger"
		txt = fmt.Sprintf(
			"A build of %s has had a timeout! :tocathinking:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	case cbpb.Build_CANCELLED:
		clr = "warning"
		txt = fmt.Sprintf(
			"A build of %s was manually canceled. :no_good:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	case cbpb.Build_EXPIRED:
		clr = "warning"
		txt = fmt.Sprintf(
			"A build of %s has expired. :headstone:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	case cbpb.Build_PENDING:
		clr = "#bab8b8"
		txt = fmt.Sprintf(
			"A build of %s needs to be approved. :vertical_traffic_light:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	default:
		clr = "warning"
		txt = fmt.Sprintf(
			"A new build of %s has completed with an unexpected status! :tocathinking:\nBranch:%s\nCommit:%s",
			triggerName,
			branchName,
			commitSha,
		)
	}

	logURL, err := notifiers.AddUTMParams(build.LogUrl, notifiers.ChatMedium)
	if err != nil {
		return nil, fmt.Errorf("failed to add UTM params: %w", err)
	}

	atch := slack.Attachment{
		Text:  txt,
		Color: clr,
		Actions: []slack.AttachmentAction{{
			Text: "View Logs",
			Type: "button",
			URL:  logURL,
		}},
	}

	return &slack.WebhookMessage{Attachments: []slack.Attachment{atch}}, nil
}
