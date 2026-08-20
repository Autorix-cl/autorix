import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Autorix Documentation",
  description: "Next-Generation Zero-Trust IAM Suite (Zanzibar ReBAC, ABAC CEL, OAuth2/OIDC, PEP Proxy, Macaroons, SAML/SCIM)",
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }]
  ],
  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'Autorix IAM',
    
    search: {
      provider: 'local',
      options: {
        locales: {
          root: {
            translations: {
              button: {
                buttonText: 'Search Documentation...',
                buttonAriaLabel: 'Search Documentation'
              },
              modal: {
                noResultsText: 'No matching documentation found',
                resetButtonTitle: 'Reset search',
                footer: {
                  selectText: 'to select',
                  navigateText: 'to navigate',
                  closeText: 'to close'
                }
              }
            }
          }
        }
      }
    },

    nav: [
      { text: 'Home', link: '/' },
      { text: 'Architecture', link: '/api_reference_and_integration_guide' },
      {
        text: 'IAM Engines',
        items: [
          { text: 'Nexus (Zanzibar ReBAC)', link: '/nexus_usage_guide' },
          { text: 'Themis (ABAC CEL Policies)', link: '/themis_usage_guide' },
          { text: 'Ego (Identity & Argon2id)', link: '/ego_usage_guide' },
          { text: 'Janus (OAuth2 & OIDC)', link: '/janus_usage_guide' },
          { text: 'Aegis (Zero-Trust PEP Proxy)', link: '/aegis_usage_guide' },
          { text: 'Vulcan (API Keys & Macaroons)', link: '/vulcan_usage_guide' },
          { text: 'Hermes (SAML 2.0 & SCIM 2.0)', link: '/hermes_usage_guide' }
        ]
      },
      {
        text: 'Client SDKs',
        items: [
          { text: 'SDKs Overview', link: '/sdk/' },
          { text: 'Go SDK Manual', link: '/sdk/go' },
          { text: 'TypeScript / React SDK', link: '/sdk/typescript' },
          { text: 'Python / FastAPI SDK', link: '/sdk/python' },
          { text: 'CLI & Direct gRPC/REST', link: '/sdk/cli' }
        ]
      },
      {
        text: 'Control Plane & UI',
        items: [
          { text: 'Argus (Fleet Control Plane)', link: '/argus_usage_guide' },
          { text: 'Console (Admin UI & Studios)', link: '/console_usage_guide' }
        ]
      },
      {
        text: 'Operations',
        items: [
          { text: 'Operations & Runbook', link: '/operations_and_runbook' },
          { text: 'Kubernetes Production Guide', link: '/production_k8s_guide' },
          { text: 'Technical Roadmap', link: '/roadmap' }
        ]
      }
    ],

    sidebar: [
      {
        text: 'Overview & Architecture',
        collapsed: false,
        items: [
          { text: 'Getting Started & Home', link: '/' },
          { text: 'Master API Reference & Architecture', link: '/api_reference_and_integration_guide' }
        ]
      },
      {
        text: 'Official Client SDKs',
        collapsed: false,
        items: [
          { text: 'SDKs Overview & Principles', link: '/sdk/' },
          { text: '🐹 Go SDK Reference', link: '/sdk/go' },
          { text: '⚛️ TypeScript / React SDK Reference', link: '/sdk/typescript' },
          { text: '🐍 Python / FastAPI SDK Reference', link: '/sdk/python' },
          { text: '💻 CLI & Universal Direct APIs', link: '/sdk/cli' }
        ]
      },
      {
        text: 'Zero-Trust IAM Engines',
        collapsed: false,
        items: [
          { text: 'Autorix Nexus (ReBAC Zanzibar)', link: '/nexus_usage_guide' },
          { text: 'Autorix Themis (ABAC CEL Engine)', link: '/themis_usage_guide' },
          { text: 'Autorix Ego (Identity & Sessions)', link: '/ego_usage_guide' },
          { text: 'Autorix Janus (OAuth2 / OIDC)', link: '/janus_usage_guide' },
          { text: 'Autorix Aegis (Reverse PEP Proxy)', link: '/aegis_usage_guide' },
          { text: 'Autorix Vulcan (Macaroon API Keys)', link: '/vulcan_usage_guide' },
          { text: 'Autorix Hermes (SAML & SCIM)', link: '/hermes_usage_guide' }
        ]
      },
      {
        text: 'Control Plane & Governance',
        collapsed: false,
        items: [
          { text: 'Autorix Argus (Fleet & Audit Trail)', link: '/argus_usage_guide' },
          { text: 'Autorix Console (Admin UI & Studios)', link: '/console_usage_guide' }
        ]
      },
      {
        text: 'Operations & Deployment',
        collapsed: false,
        items: [
          { text: 'Day-1 & Day-2 Operations Runbook', link: '/operations_and_runbook' },
          { text: 'Production Kubernetes Deployment', link: '/production_k8s_guide' },
          { text: 'Roadmap & Future Phases', link: '/roadmap' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/autorix-cl/autorix' }
    ],

    footer: {
      message: 'Released under the Apache 2.0 License. Built for Enterprise Zero-Trust Security.',
      copyright: 'Copyright © 2026 Autorix Platform Team'
    }
  }
})
