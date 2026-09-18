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
  redirects: { '/guides/customize/': '/guides/select-rules/' },
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
        { label: 'What is Code Rules?', slug: 'overview' },
        { label: 'Install Code Rules', slug: 'guides/install' },
        { label: 'Set up a project', slug: 'guides/set-up-project' },
        { label: 'Use rules in a project', slug: 'guides/use-rules' },
        { label: 'Project status', slug: 'status' },
      ] },
      { label: 'Concepts', items: [
        { label: 'Rule', slug: 'concepts/rule' },
        { label: 'Group', slug: 'concepts/groups' },
        { label: 'Library', slug: 'concepts/libraries' },
      ] },
      { label: 'Guides', items: [
        { label: 'Import rules', slug: 'guides/select-rules' },
        { label: 'Create a rule library', slug: 'guides/create-library' },
        { label: 'Make a rule', slug: 'guides/write-rules' },
        { label: 'Adapt a third-party rule', slug: 'guides/adapt-rules' },
        { label: 'Update rules', slug: 'guides/update' },
        { label: 'Conflicting guidance', slug: 'guides/conflicting-guidance' },
        { label: 'License rules', slug: 'guides/license-rules' },
      ] },
      { label: 'Reference', items: [
        { label: 'Configuration', slug: 'reference/configuration' },
        { label: 'Files and formats', slug: 'reference/files' },
        { label: 'Rule rubric and template', slug: 'reference/rule-authoring' },
        { label: 'CLI commands', slug: 'reference/cli' },
        { label: 'How imports work', slug: 'reference/imports' },
        { label: 'Sync and recovery', slug: 'reference/sync' },
      ] },
      { label: 'For agents', items: [{ label: 'Plan, write, and review', slug: 'for-agents' }] },
    ],
    tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 3 },
  })],
});
