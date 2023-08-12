__webpack_public_path__ = "/docs/"

module.exports = {
  title: 'vcluster docs | Virtual Clusters for Kubernetes',
  tagline: 'Virtual Clusters for Kubernetes',
  url: 'https://vcluster.com',
  baseUrl: __webpack_public_path__,
  favicon: '/media/vcluster_symbol.svg',
  organizationName: 'loft-sh', // Usually your GitHub org/user name.
  projectName: 'vcluster', // Usually your repo name.
  themeConfig: {
    colorMode: {
      defaultMode: 'light',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      logo: {
        alt: 'vcluster',
        src: '/media/vcluster_Horizontal_MonoBranding.svg',
        href: 'https://vcluster.com/',
        target: '_self',
      },
      items: [
        {
          href: 'https://vcluster.com/',
          label: 'Website',
          position: 'left',
          target: '_self'
        },
        {
          to: '/docs/what-are-virtual-clusters',
          label: 'Docs',
          position: 'left'
        },
        {
          href: 'https://loft.sh/blog',
          label: 'Blog',
          position: 'left',
          target: '_self'
        },
        {
          href: 'https://slack.loft.sh/',
          className: 'slack-link',
          'aria-label': 'Slack',
          position: 'right',
        },
        {
          href: 'https://github.com/loft-sh/vcluster',
          className: 'github-link',
          'aria-label': 'GitHub',
          position: 'right',
        },
      ],
    },
    algolia: {
      appId: "K85RIQNFGF",
      apiKey: "42375731adc726ebb99849e9051aa9b4",
      indexName: "vcluster",
      placeholder: "Search...",
      algoliaOptions: {}
    },
    footer: {
      style: 'light',
      links: [],
      copyright: `Copyright © ${new Date().getFullYear()} <a href="https://loft.sh/">Loft Labs, Inc.</a>`,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          path: 'pages',
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl:
            'https://github.com/loft-sh/vcluster/edit/main/docs/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
  plugins: [],
  scripts: [
    {
      src:
        'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js',
      async: true,
    },
    {
      src:
        '/docs/js/custom.js',
      async: true,
    },
  ],
};
