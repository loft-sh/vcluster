/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

module.exports = {
  adminSidebar: [
    {
      type: 'doc',
      id: 'what-are-virtual-clusters',
    },
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        {
          type: 'category',
          label: 'Quickstart',
          collapsed: false,
          items: [
            'getting-started/setup',
            'getting-started/deployment',
            'getting-started/connect',
            'getting-started/cleanup',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      collapsed: true,
      items: [
        'architecture/overview',
        {
          type: 'category',
          label: 'Control Plane',
          collapsed: true,
          items: [
            'architecture/control_plane/control_plane',
            'architecture/control_plane/k8s_distros',
            'architecture/control_plane/isolated_control_planes',
          ],
        },
        {
          type: 'category',
          label: 'Syncer',
          collapsed: true,
          items: [
            'architecture/syncer/syncer',
            'architecture/syncer/single_vs_multins',

          ],
        },
        'architecture/scheduling',
        'architecture/nodes',
      ],
    },
    {
      type: 'category',
      label: 'Networking',
      collapsed: true,
      items: [
        'networking/networking',
        'networking/coreDNS',
        'networking/integrated_coredns',
        {
          type: 'category',
          label: 'Mapping Traffic',
          collapsed: true,
          items: [
            'networking/internal_traffic/host_to_vcluster',
            'networking/internal_traffic/vcluster_to_host',
          ],
        },
        'networking/ingress_traffic',
        'networking/network_policies',
      ],
    },
    {
      type: 'category',
      label: 'Sync',
      collapsed: true,
      items: [
        'syncer/core_resources',
        {
          type: "category",
          label: "Syncer",
          collapsed: true,
          items: [
            'syncer/config',
          ]
        },
        {
          type: "category",
          label: "Other resources",
          collapsed: true,
          items: [
            'syncer/other_resources/overview',
            'syncer/other_resources/generic_sync',
            'syncer/other_resources/config_syntax',
            'syncer/other_resources/multi_namespace_mode',
          ]
        },
        'syncer/generic_resource_patches',
        {
          type: "category",
          label: "Plugins",
          collapsed: true,
          items: [
              'plugins/overview',
              'plugins/tutorial',
          ]
        },
      ],
    },
    {
      type: 'category',
      label: 'Using vclusters',
      collapsed: true,
      items: [
        {
          type: 'category',
          label: 'Accessing vcluster',
          collapsed: true,
          items: [
            'using-vclusters/kube-context',
            'using-vclusters/access',
          ],
        },
        'using-vclusters/pausing-vcluster',
        'using-vclusters/backup-restore',
      ],
    },
    {
      type: 'category',
      label: 'Deploying vclusters',
      collapsed: true,
      items: [
        {
          type: 'category',
          label: 'Kubernetes Distros',
          collapsed: true,
          items: [
            'deploying-vclusters/supported-distros',
          ],
        },
        {
          type: 'category',
          label: 'Persistent vclusters',
          collapsed: true,
          items: [
            'deploying-vclusters/persistence',
          ],
        },
        'deploying-vclusters/high-availability',
        {
          type: 'category',
          label: 'On Init',
          collapsed: true,
          items: [
            'deploying-vclusters/init-manifests',
            'deploying-vclusters/init-charts',
          ],
        },
        {
          type: 'category',
          label: 'Integrations',
          collapsed: true,
          items: [
            'deploying-vclusters/integrations-openshift',
          ],
        },
      ],
    },
    {
      type: 'doc',
      id: 'storage',
    },
    {
      type: 'category',
      label: 'Observability',
      collapsed: true,
      items: [
        {
          type: 'category',
          label: 'Collecting Metrics',
          collapsed: true,
          items: [
            'o11y/metrics/metrics_server_proxy',
            'o11y/metrics/metrics_server',
            'o11y/metrics/monitoring_vcluster',
          ]
        },
        {
          type: 'category',
          label: 'Logging',
          collapsed: true,
          items: [
            'o11y/logging/hpm',
            'o11y/logging/central_hpm',
            'o11y/logging/elk_stack',
            'o11y/logging/grafana_loki',
          ]
        }
      ]
    },
    {
      type: 'category',
      label: 'Security',
      collapsed: true,
      items: [
        'security/rootless-mode',
        'security/isolated-mode',
        'security/admission-control',
        'security/quotas-limits',
        'security/pod-security',
        'security/network-isolation',
        'security/other-topics',
      ],
    },
    {
      type: 'category',
      label: 'Advanced topics',
      collapsed: true,
      items: [
        {
          type: 'category',
          label: 'Plugins',
          collapsed: true,
          items: [
            'advanced-topics/plugins-overview',
            'advanced-topics/plugins-development'
          ],
        },
        'advanced-topics/telemetry',
      ],
    },
    {
      type: 'category',
      label: 'Help and Troubleshooting',
      collapsed: true,
      items: [
        'troubleshooting',
        'community'
      ]
    },
    {
      type: 'doc',
      id: 'config-reference',
    },
    {
      type: 'link',
      label: 'Originally created by Loft',
      href: 'https://loft.sh/',
    },
  ],
};
