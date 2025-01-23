__webpack_public_path__ = "/docs/v0.19/";

module.exports = {
  title: "vcluster docs | Virtual Clusters for Kubernetes",
  tagline: "Virtual Clusters for Kubernetes",
  url: "https://vcluster.com",
  baseUrl: __webpack_public_path__,
  favicon: "/media/vcluster_symbol.svg",
  organizationName: "loft-sh", // Usually your GitHub org/user name.
  projectName: "vcluster", // Usually your repo name.
  themeConfig: {
    colorMode: {
      defaultMode: "light",
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    navbar: {
      logo: {
        alt: "vcluster",
        src: "/media/vCluster_horizontal-orange.svg",
        href: "https://vcluster.com/",
        target: "_self",
      },
      items: [
        {
          type: "docsVersionDropdown",
          position: "left",
          dropdownItemsAfter: [
            { to: "https://vcluster.com/docs", label: "Latest" },
          ],
          dropdownActiveClassDisabled: true,
        },
        {
          href: "https://vcluster.com/",
          label: "Website",
          position: "left",
          target: "_self",
        },
        {
          label: "Docs",
          position: "left",
          to: "/",
        },
        {
          href: "https://loft.sh/blog",
          label: "Blog",
          position: "left",
          target: "_self",
        },
        {
          href: "https://slack.loft.sh/",
          className: "slack-link",
          "aria-label": "Slack",
          position: "right",
        },
        {
          href: "https://github.com/loft-sh/vcluster",
          className: "github-link",
          "aria-label": "GitHub",
          position: "right",
        },
      ],
    },
    footer: {
      style: "light",
      links: [],
      copyright: `Copyright © ${new Date().getFullYear()} <a href="https://loft.sh/">Loft Labs, Inc.</a>`,
    },
    announcementBar: {
      id: 'version-info',
      content: '<strong>You are viewing docs for <code>vCluster</code> v0.19. <a href="https://www.vcluster.com/docs" target="_blank">See latest releases.</a></strong>',
      backgroundColor: '#3c7a89',
      textColor: '#ffffff',
      isCloseable: true,
    },
  },
  presets: [
    [
      "@docusaurus/preset-classic",
      {
        docs: {
          path: "pages",
          routeBasePath: "/",
          sidebarPath: require.resolve("./sidebars.js"),
          editUrl: "https://github.com/loft-sh/vcluster/edit/main/docs/",
          versions: {
            current: {
              label: "Legacy",
            },
          },
        },
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
      },
    ],
  ],
  scripts: [
    {
      src: "https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js",
      async: true,
    },
    {
      src: "/docs/v0.19/js/custom.js",
      async: true,
    },
  ],
};
