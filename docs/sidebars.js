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
        'architecture/basics',
        'architecture/scheduling',
        'architecture/networking',
        'architecture/storage',
        'architecture/nodes',
        'architecture/synced-resources',
      ],
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
      type: "category",
      label: "Plugins",
      collapsed: false,
      items: [
          'plugins/overview',
          'plugins/tutorial',
          'plugins/generic-crd-sync',
      ]
    },
    {
      type: 'doc',
      id: 'troubleshooting',
    },
    {
      type: 'doc',
      id: 'config-reference',
    },
    {
      type: 'category',
      label: 'CLI Reference',
      items: [
      ],
    },
    {
      type: 'link',
      label: 'Originally created by Loft',
      href: 'https://loft.sh/',
    },
  ],
};
