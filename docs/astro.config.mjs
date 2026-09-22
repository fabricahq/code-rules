/** @fileoverview Configures the Code Rules documentation site, navigation, and Markdown rendering. */

import { defineConfig } from 'astro/config';
import { unified } from '@astrojs/markdown-remark';
import starlight from '@astrojs/starlight';
import tailwindcss from '@tailwindcss/vite';
import accessibleAsideTitles from './src/plugins/accessible-aside-titles.mjs';

export default defineConfig({
  devToolbar: { enabled: false },
  // Keep native bindings outside the SSR bundle: https://vite.dev/config/ssr-options.html#ssr-external
  // Satteri's installed index.js resolves platform binaries relative to its package.
  vite: { plugins: [tailwindcss()], ssr: { external: ['satteri'] } },
  redirects: {
    '/guides/customize/': '/guides/select-rules/',
    '/overview/': '/start-here/overview/',
    '/status/': '/start-here/install/',
    '/guides/install/': '/start-here/install/',
    '/guides/set-up-project/': '/start-here/set-up-project/',
    '/guides/use-rules/': '/start-here/set-up-project/',
    '/start-here/use-rules/': '/start-here/set-up-project/',
    '/guides/create-library/': '/start-here/create-library/',
  },
  markdown: { processor: unified({ rehypePlugins: [accessibleAsideTitles] }) },
  integrations: [starlight({
    title: 'Code Rules',
    description: 'The package manager for your engineering rules',
    favicon: '/favicon.svg',
    disable404Route: true,
    customCss: ['./src/styles/tailwind.css', './src/styles/custom.css', './src/styles/home.css'],
    components: {
      Hero: './src/components/HomePage.astro',
      SiteTitle: './src/components/SiteTitle.astro',
      SocialIcons: './src/components/NavLinks.astro',
      ThemeSelect: './src/components/ThemeSelect.astro',
      Footer: './src/components/Footer.astro',
    },
    sidebar: [
      { label: 'Start here', items: [
        { label: 'What is Code Rules?', slug: 'start-here/overview' },
        { label: 'Install Code Rules', slug: 'start-here/install' },
        { label: 'Set up your first project', slug: 'start-here/set-up-project' },
        { label: 'Create your first library', slug: 'start-here/create-library' },
      ] },
      { label: 'Concepts', items: [
        { label: 'Rule', slug: 'concepts/rule' },
        { label: 'Group', slug: 'concepts/groups' },
        { label: 'Library', slug: 'concepts/libraries' },
        { label: 'Project', slug: 'concepts/project' },
      ] },
      { label: 'Guides', items: [
        { label: 'Manage project rules', items: [
          { label: 'Import and customize rules', slug: 'guides/select-rules' },
          { label: 'Resolve conflicting rules', slug: 'guides/conflicting-guidance' },
          { label: 'Update rules', slug: 'guides/update' },
        ] },
        { label: 'Write and share rules', items: [
          { label: 'Write a rule', slug: 'guides/write-rules' },
          { label: 'License rules', slug: 'guides/license-rules' },
          { label: 'Adapt a third-party rule', slug: 'guides/adapt-rules' },
        ] },
      ] },
      { label: 'Reference', items: [
        { label: 'Projects', items: [
          { label: 'Project files', slug: 'reference/files' },
          { label: 'Configuration', slug: 'reference/configuration' },
          { label: 'How imports work', slug: 'reference/imports' },
          { label: 'Sync and recovery', slug: 'reference/sync' },
          { label: 'Provenance', slug: 'reference/provenance' },
        ] },
        { label: 'Rules and libraries', items: [
          { label: 'Rule and library format', slug: 'reference/rule-library-format' },
          { label: 'Rule rubric and template', slug: 'reference/rule-authoring' },
        ] },
        { label: 'CLI commands', slug: 'reference/cli' },
      ] },
      { label: 'For agents', items: [{ label: 'Plan, write, and review', slug: 'for-agents' }] },
    ],
    tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 3 },
  })],
});
