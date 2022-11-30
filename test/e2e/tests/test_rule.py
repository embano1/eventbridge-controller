# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the EventBridge Rule resource
"""

import boto3
import logging
import time
from typing import Dict

import pytest

from acktest import tags
from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_eventbridge_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import RuleValidator

RESOURCE_PLURAL = "rules"

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def simple_event_bus():
    resource_name = random_suffix_name("eventbridge-bus", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["EVENT_BUS_NAME"] = resource_name

    resource_data = load_eventbridge_resource(
        "event_bus",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Get latest event_bus CR
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@pytest.fixture
def simple_rule(simple_event_bus):
    resource_name = random_suffix_name("eventbridge-rule", 24)
    _, eb_cr = simple_event_bus

    replacements = REPLACEMENT_VALUES.copy()
    replacements["EVENT_BUS_NAME"] = eb_cr["spec"]["name"]
    replacements["RULE_NAME"] = resource_name
    replacements["SCHEDULE_EXPRESSION"] = "rate(1 minute)"
    replacements["EVENT_PATTERN"] = "{\"detail-type\":\"ack-event\"}"

    resource_data = load_eventbridge_resource(
        "rule",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Get latest rule CR
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestRule:
    def test_create_delete_with_tags(self, eventbridge_client, simple_rule):
        (ref, cr) = simple_rule

        rule_name = cr["spec"]["name"]
        rule_arn = cr["status"]["ackResourceMetadata"]["arn"]

        eventbridge_validator = RuleValidator(eventbridge_client)
        # verify that rule exists
        assert eventbridge_validator.get_rule(rule_name) is not None

        # verify that rule tags are created
        rule_tags = eventbridge_validator.list_resource_tags(rule_arn)
        tags.assert_ack_system_tags(
            tags=rule_tags,
        )
        tags_dict = tags.to_dict(
            cr["spec"]["tags"],
        )
        tags.assert_equal_without_ack_tags(
            actual = tags_dict,
            expected = rule_tags,
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check rule doesn't exist
        assert eventbridge_validator.get_rule(rule_name) is None
    
    # def test_create_delete_with_targets

    def test_rule_update_targets(self, eventbridge_client, simple_rule):
        (ref, cr) = simple_rule

        rule_name = cr["spec"]["name"]
        rule_arn = cr["status"]["ackResourceMetadata"]["arn"]

        eventbridge_validator = RuleValidator(eventbridge_client)
        # verify that rule exists
        assert eventbridge_validator.get_rule(rule_name) is not None


        # Play with and test target update operations

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check rule doesn't exist
        assert eventbridge_validator.get_rule(rule_name) is None