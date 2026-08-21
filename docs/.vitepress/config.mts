import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Autorix Documentation",
  description: "Next-Generation Zero-Trust IAM Suite (Zanzibar ReBAC, ABAC CEL, OAuth2/OIDC, PEP Proxy, Macaroons, SAML/SCIM)",
  markdown: {
    lineNumbers: true
  },
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }]
  ],
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
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
        docFooter: {
          prev: 'Previous Page',
          next: 'Next Page'
        }
      }
    },
    es: {
      label: 'Español',
      lang: 'es',
      link: '/es/',
      title: "Documentación de Autorix",
      description: "Suite de Identidad y Control de Acceso Zero-Trust de Próxima Generación",
      themeConfig: {
        nav: [
          { text: 'Inicio', link: '/es/' },
          { text: 'Arquitectura', link: '/es/api_reference_and_integration_guide' },
          {
            text: 'Motores IAM',
            items: [
              { text: 'Nexus (Zanzibar ReBAC)', link: '/es/nexus_usage_guide' },
              { text: 'Themis (Políticas ABAC CEL)', link: '/es/themis_usage_guide' },
              { text: 'Ego (Identidad y Argon2id)', link: '/es/ego_usage_guide' },
              { text: 'Janus (OAuth2 y OIDC)', link: '/es/janus_usage_guide' },
              { text: 'Aegis (Proxy PEP Zero-Trust)', link: '/es/aegis_usage_guide' },
              { text: 'Vulcan (API Keys y Macaroons)', link: '/es/vulcan_usage_guide' },
              { text: 'Hermes (SAML 2.0 y SCIM 2.0)', link: '/es/hermes_usage_guide' }
            ]
          },
          {
            text: 'SDKs Oficiales',
            items: [
              { text: 'Visión General', link: '/es/sdk/' },
              { text: 'Manual SDK Go', link: '/es/sdk/go' },
              { text: 'SDK TypeScript / React', link: '/es/sdk/typescript' },
              { text: 'SDK Python / FastAPI', link: '/es/sdk/python' },
              { text: 'CLI y REST/gRPC Directo', link: '/es/sdk/cli' }
            ]
          },
          {
            text: 'Plano de Control y UI',
            items: [
              { text: 'Argus (Plano de Control)', link: '/es/argus_usage_guide' },
              { text: 'Console (Panel de Administración)', link: '/es/console_usage_guide' }
            ]
          },
          {
            text: 'Operaciones',
            items: [
              { text: 'Runbook de Operaciones', link: '/es/operations_and_runbook' },
              { text: 'Guía de Kubernetes', link: '/es/production_k8s_guide' },
              { text: 'Hoja de Ruta (Roadmap)', link: '/es/roadmap' }
            ]
          }
        ],
        sidebar: [
          {
            text: 'Visión General y Arquitectura',
            collapsed: false,
            items: [
              { text: 'Primeros Pasos e Inicio', link: '/es/' },
              { text: 'Guía Maestra de Arquitectura', link: '/es/api_reference_and_integration_guide' }
            ]
          },
          {
            text: 'SDKs Oficiales',
            collapsed: false,
            items: [
              { text: 'Visión General y Principios', link: '/es/sdk/' },
              { text: '🐹 Referencia SDK de Go', link: '/es/sdk/go' },
              { text: '⚛️ Referencia SDK de TypeScript', link: '/es/sdk/typescript' },
              { text: '🐍 Referencia SDK de Python', link: '/es/sdk/python' },
              { text: '💻 CLI y APIs Universales', link: '/es/sdk/cli' }
            ]
          },
          {
            text: 'Motores Zero-Trust',
            collapsed: false,
            items: [
              { text: 'Autorix Nexus (ReBAC Zanzibar)', link: '/es/nexus_usage_guide' },
              { text: 'Autorix Themis (Políticas ABAC CEL)', link: '/es/themis_usage_guide' },
              { text: 'Autorix Ego (Identidad y Sesiones)', link: '/es/ego_usage_guide' },
              { text: 'Autorix Janus (OAuth2 / OIDC)', link: '/es/janus_usage_guide' },
              { text: 'Autorix Aegis (Proxy PEP Inverso)', link: '/es/aegis_usage_guide' },
              { text: 'Autorix Vulcan (Macaroon API Keys)', link: '/es/vulcan_usage_guide' },
              { text: 'Autorix Hermes (SAML y SCIM)', link: '/es/hermes_usage_guide' }
            ]
          },
          {
            text: 'Gobernanza y Control',
            collapsed: false,
            items: [
              { text: 'Autorix Argus (Flota y Auditoría)', link: '/es/argus_usage_guide' },
              { text: 'Autorix Console (Panel y Studios)', link: '/es/console_usage_guide' }
            ]
          },
          {
            text: 'Operaciones y Despliegue',
            collapsed: false,
            items: [
              { text: 'Runbook Operativo Día 1 y 2', link: '/es/operations_and_runbook' },
              { text: 'Despliegue en Kubernetes', link: '/es/production_k8s_guide' },
              { text: 'Hoja de Ruta (Roadmap)', link: '/es/roadmap' }
            ]
          }
        ],
        docFooter: {
          prev: 'Página Anterior',
          next: 'Página Siguiente'
        },
        outline: {
          label: 'En esta página'
        }
      }
    }
  },
  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'Autorix IAM',
    outline: {
      level: [2, 3],
      label: 'On this page'
    },
    search: {
      provider: 'local',
      options: {
        locales: {
          es: {
            translations: {
              button: {
                buttonText: 'Buscar en la documentación...',
                buttonAriaLabel: 'Buscar en la documentación'
              },
              modal: {
                noResultsText: 'No se encontraron resultados',
                resetButtonTitle: 'Limpiar búsqueda',
                footer: {
                  selectText: 'para seleccionar',
                  navigateText: 'para navegar',
                  closeText: 'para cerrar'
                }
              }
            }
          }
        }
      }
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/autorix-cl/autorix' }
    ],
    footer: {
      message: 'Released under Apache 2.0 License. Built for Enterprise Zero-Trust Security.',
      copyright: 'Copyright © 2026 Autorix Platform Team'
    }
  }
})
