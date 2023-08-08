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
          type: 'doc',
          id: 'quickstart',
        },
        {
          type: 'category',
          label: 'Full Guide',
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
      collapsed: false,
      items: [
        'architecture/overview',
        {
          type: 'category',
          label: 'Control Plane',
          collapsed: false,
          items: [
            'architecture/control_plane/control_plane',
            'architecture/control_plane/k8s_distros',
            'architecture/control_plane/isolated_control_planes',
          ],
        },
        {
          type: 'category',
          label: 'Syncer',
          collapsed: false,
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
      collapsed: false,
      items: [
        'networking/networking',
        'networking/coreDNS',
        {
          type: 'category',
          label: 'Mapping Traffic',
          collapsed: false,
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
      collapsed: false,
      items: [
        'syncer/core_resources',
        {
          type: "category",
          label: "Syncer",
          collapsed: false,
          items: [
            'syncer/config',
          ]
        },
        {
          type: "category",
          label: "Other resources",
          collapsed: false,
          items: [
            'syncer/other_resources/overview',
            'syncer/other_resources/generic_sync',
            'syncer/other_resources/config_syntax',
            'syncer/other_resources/multi_namespace_mode',
          ]
        },
        {
          type: "category",
          label: "Plugins",
          collapsed: false,
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
      collapsed: false,
      items: [
        {
          type: 'category',
          label: 'Accessing vcluster',
          collapsed: false,
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
      type: 'doc',
      id: 'storage',
    },
    {
      type: 'category',
      label: 'Operator Guide',
      collapsed: false,
      items: [
        'operator/external-access',
        'operator/external-datastore',
        'operator/accessing-vcluster',
        'operator/init-manifests',
        'operator/monitoring-logging',
        'operator/high-availability',
        'operator/other-distributions',
        'operator/restricted-hosts',
        'operator/pausing-vcluster',
        'operator/backup',
        'operator/security',
        'operator/cluster-api-provider',
      ],
    },
    {
      type: 'doc',
      id: 'troubleshooting',
    },
    {
      type: 'doc',
      id: 'telemetry',
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
