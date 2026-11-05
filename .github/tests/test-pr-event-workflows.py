#!/usr/bin/env python3
"""Regression checks for PR workflow trust boundaries."""
from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / '.github' / 'workflows'
SOURCE = '.github/workflows/pull-request-events.yaml'
HEAD_GUARD = 'github.event.pull_request.head.repo.full_name == github.repository'
FORBIDDEN_TRIGGER = 'pull_request' + '_target'


def workflow(name):
    return yaml.load((WORKFLOWS / name).read_text(), Loader=yaml.BaseLoader)


class WorkflowTrustTests(unittest.TestCase):
    def test_no_target_triggers(self):
        for path in WORKFLOWS.iterdir():
            if path.suffix not in ('.yaml', '.yml'):
                continue
            data = yaml.load(path.read_text(), Loader=yaml.BaseLoader)
            with self.subTest(path=path.name):
                self.assertNotIn(FORBIDDEN_TRIGGER, data.get('on', {}))
                self.assertNotIn(FORBIDDEN_TRIGGER, path.read_text())

    def test_signal_has_no_privilege_or_untrusted_outputs(self):
        signal = workflow('pull-request-events.yaml')
        self.assertEqual(set(signal['on']), {'pull_request'})
        self.assertEqual(set(signal['on']['pull_request']['types']), {'labeled'})
        self.assertEqual(signal['permissions'], {})
        self.assertEqual(set(signal['jobs']), {'relay'})
        relay = signal['jobs']['relay']
        self.assertEqual(relay['permissions'], {})
        self.assertEqual(len(relay['steps']), 1)
        self.assertEqual(relay['steps'][0]['run'], ':')
        self.assertNotIn('uses', relay['steps'][0])
        self.assertNotIn('secrets.', (WORKFLOWS / 'pull-request-events.yaml').read_text())

    def test_trusted_backport_resolves_and_forwards_only_verified_events(self):
        backport = workflow('backport.yaml')
        self.assertEqual(backport['permissions'], {})
        self.assertEqual(backport['on']['push']['branches'], ['main'])
        self.assertEqual(backport['on']['workflow_run']['workflows'], ['Pull request events'])
        self.assertEqual(backport['on']['workflow_run']['types'], ['completed'])
        event, call = backport['jobs']['event'], backport['jobs']['backport']
        condition = event['if']
        for clause in ("github.ref == format('refs/heads/{0}', github.event.repository.default_branch)",
                       "endsWith(github.workflow_ref, format('@refs/heads/{0}', github.event.repository.default_branch))",
                       "github.event_name == 'push'", "github.event.workflow_run.conclusion == 'success'",
                       "github.event.workflow_run.event == 'pull_request'"):
            self.assertIn(clause, condition)
        readonly = {'actions': 'read', 'contents': 'read', 'pull-requests': 'read'}
        self.assertEqual(event['permissions'], readonly)
        self.assertEqual(call['permissions'], readonly)
        step = event['steps'][0]
        self.assertEqual(step['with']['source-workflow'], SOURCE)
        self.assertEqual(call['with']['source-workflow'], SOURCE)
        self.assertEqual(event['outputs']['events-json'], '${{ steps.event.outputs.events-json }}')
        self.assertEqual(call['needs'], 'event')
        self.assertEqual(call['if'], "needs.event.outputs.eligible == 'true'")
        self.assertEqual(call['strategy']['matrix']['event'], '${{ fromJSON(needs.event.outputs.events-json) }}')
        self.assertEqual(call['with']['pr-event'], '${{ toJSON(matrix.event) }}')
        resolver = step['uses'].rsplit('@', 1)[1]
        reusable = call['uses'].rsplit('@', 1)[1]
        self.assertRegex(resolver, r'^[0-9a-f]{40}$')
        self.assertEqual(resolver, reusable)

    def test_pr_code_jobs_are_read_only_and_drop_checkout_credentials(self):
        for name in ('e2e.yaml', 'e2e-ginkgo.yaml', 'unit-tests.yaml', 'lint.yaml'):
            data = workflow(name)
            with self.subTest(workflow=name):
                self.assertEqual(data['permissions'], {'contents': 'read'})
                for job_name, job in data['jobs'].items():
                    for step in job.get('steps', []):
                        if step.get('uses', '').startswith('actions/checkout@'):
                            self.assertEqual(step.get('with', {}).get('persist-credentials'), 'false', job_name)

    def test_privileged_pr_jobs_require_same_repository(self):
        for name, job in (('auto-approve-bot-prs.yaml', 'auto-approve'),
                          ('conflict-check.yaml', 'conflict-check'),
                          ('clean-github-cache.yaml', 'clean-github-cache'),
                          ('actionlint.yml', 'actionlint')):
            data = workflow(name)
            with self.subTest(workflow=name):
                self.assertEqual(data['permissions'], {})
                self.assertIn(HEAD_GUARD, data['jobs'][job]['if'])
        ginkgo = workflow('e2e-ginkgo.yaml')['jobs']['ginkgo-e2e-tests']
        self.assertIn(HEAD_GUARD, ginkgo['if'])
        self.assertIn("github.event_name != 'pull_request'", ginkgo['if'])
        self.assertIn("github.event_name == 'workflow_dispatch'", ginkgo['if'])
        lint = workflow('lint.yaml')['jobs']['golangci']
        secret_steps = [s for s in lint['steps'] if 'secrets.' in str(s)]
        self.assertTrue(secret_steps)
        for step in secret_steps:
            self.assertIn(HEAD_GUARD, step['if'])


if __name__ == '__main__':
    unittest.main()
