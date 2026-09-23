/** @fileoverview Plays the homepage's tabbed agent transcripts; pauses for readers, hidden pages, and reduced-motion preferences. */
import { TranscriptPlayback } from './transcript-playback';

/** Cached, server-rendered content avoids replacing highlighted markup while the transcript streams. */
type Entry = {
  element: HTMLElement;
  note: HTMLElement;
  characters: string[];
  tool: HTMLElement;
  lines: HTMLElement[];
};

/**
 * Connects transcript playback to the visible session, browser clock, and controls.
 * Initial views are static. Play/Replay explicitly opt into animation, even with
 * reduced motion; preference changes and Show full session stop playback.
 * It never runs the illustrated commands.
 */
class AgentDemo extends HTMLElement {
  private entries = new Map<HTMLElement, Entry[]>();
  private panel: HTMLElement | undefined;
  private playback: TranscriptPlayback | undefined;
  private isVisible = false;
  private timer: ReturnType<typeof setTimeout> | undefined;
  private lastTick = 0;
  private events: AbortController | undefined;
  private observer: IntersectionObserver | undefined;
  private motion = window.matchMedia('(prefers-reduced-motion: reduce)');

  connectedCallback() {
    if (this.events) return;
    this.events = new AbortController();
    const { signal } = this.events;
    this.cacheTranscripts();
    this.panel =
      this.querySelector<HTMLElement>('[role="tabpanel"]:not([hidden])') ??
      undefined;
    this.playback ??= new TranscriptPlayback(
      new Map(
        Array.from(this.entries, ([panel, entries]) => [
          panel.id,
          entries.map((entry) => ({
            characters: entry.characters.length,
            lines: entry.lines.length,
          })),
        ]),
      ),
      this.panel?.id ?? '',
    );
    this.querySelectorAll<HTMLElement>('[data-tabs], [data-controls]').forEach(
      (element) => {
        element.hidden = false;
      },
    );
    this.addEventListener('click', (event) => this.onClick(event), { signal });
    this.addEventListener('keydown', (event) => this.onKeydown(event), {
      signal,
    });
    this.addEventListener(
      'focusin',
      (event) => {
        if (
          event.target instanceof Element &&
          event.target.closest('[data-transcript]') &&
          event.target.matches(':focus-visible')
        )
          this.pause();
      },
      { signal },
    );
    document.addEventListener('visibilitychange', () => this.schedule(), {
      signal,
    });
    this.motion.addEventListener('change', () => this.showComplete(), {
      signal,
    });
    this.observer = new IntersectionObserver(
      ([entry]) => {
        this.isVisible = entry?.isIntersecting ?? false;
        this.schedule();
      },
      { threshold: 0.15 },
    );
    this.observer.observe(this);
    this.showComplete();
  }

  disconnectedCallback() {
    clearTimeout(this.timer);
    this.observer?.disconnect();
    this.events?.abort();
    this.events = undefined;
    // Restore source text so reconnecting this element never caches a partially typed message.
    for (const entries of this.entries.values()) {
      for (const entry of entries)
        entry.note.textContent = entry.characters.join('');
    }
    this.entries.clear();
  }

  private cacheTranscripts() {
    for (const panel of this.querySelectorAll<HTMLElement>(
      '[role="tabpanel"]',
    )) {
      const entries: Entry[] = [];
      for (const element of panel.querySelectorAll<HTMLElement>(
        '[data-event]',
      )) {
        const note = element.querySelector<HTMLElement>('[data-note]');
        const tool = element.querySelector<HTMLElement>('[data-tool]');
        if (!note || !tool) continue;
        const lines = Array.from(tool.querySelectorAll<HTMLElement>('.line'));
        lines.forEach((line) => {
          if (tool.dataset.kind !== 'PATCH') return;
          line.classList.toggle(
            'added',
            line.textContent?.startsWith('+') ?? false,
          );
          line.classList.toggle(
            'removed',
            line.textContent?.startsWith('-') ?? false,
          );
        });
        entries.push({
          element,
          note,
          tool,
          lines,
          characters: Array.from(note.textContent ?? ''),
        });
      }
      this.entries.set(panel, entries);
    }
  }

  private onClick(event: MouseEvent) {
    if (!(event.target instanceof Element)) return;
    const button = event.target.closest<HTMLButtonElement>('button');
    if (!button) {
      if (
        !event.target.closest('[data-session-window]') ||
        event.target.closest('a, input, textarea, select, [contenteditable]') ||
        this.playback?.view.isPreview ||
        this.playback?.view.isComplete
      )
        return;
      if (!window.getSelection()?.isCollapsed) {
        this.pause();
        return;
      }
      this.togglePlayback();
      return;
    }
    if (button.hasAttribute('data-tab')) this.selectTab(button);
    else if (button.hasAttribute('data-start')) {
      if (this.playback?.view.isPaused) this.resume();
      else this.restart();
      this.querySelector<HTMLButtonElement>('[data-play]')?.focus({
        preventScroll: true,
      });
    } else if (button.hasAttribute('data-replay')) this.restart();
    else if (button.hasAttribute('data-finish')) {
      this.showComplete(true);
    } else if (button.hasAttribute('data-play')) {
      if (this.playback?.view.isComplete) {
        this.restart();
        return;
      }
      this.togglePlayback();
    }
  }

  private onKeydown(event: KeyboardEvent) {
    if (
      !(event.target instanceof HTMLElement) ||
      !event.target.matches('[data-tab]')
    )
      return;
    const tabs = Array.from(
      this.querySelectorAll<HTMLButtonElement>('[data-tab]'),
    );
    const current = tabs.findIndex((tab) => tab === event.target);
    const next =
      event.key === 'ArrowRight'
        ? (current + 1) % tabs.length
        : event.key === 'ArrowLeft'
          ? (current + tabs.length - 1) % tabs.length
          : event.key === 'Home'
            ? 0
            : event.key === 'End'
              ? tabs.length - 1
              : undefined;
    const tab = next === undefined ? undefined : tabs[next];
    if (!tab) return;
    event.preventDefault();
    this.selectTab(tab);
    tab.focus({ preventScroll: true });
  }

  private selectTab(selected: HTMLButtonElement) {
    if (selected.getAttribute('aria-selected') === 'true') return;
    for (const tab of this.querySelectorAll<HTMLButtonElement>('[data-tab]')) {
      tab.setAttribute('aria-selected', String(tab === selected));
      tab.tabIndex = tab === selected ? 0 : -1;
    }
    for (const panel of this.entries.keys()) {
      panel.hidden = panel.id !== selected.getAttribute('aria-controls');
      if (!panel.hidden) this.panel = panel;
    }
    if (this.panel) this.playback?.select(this.panel.id);
    this.render();
    this.schedule();
  }

  private restart() {
    // Reserve the full window before hiding content for playback.
    const sessionWindow = this.querySelector<HTMLElement>(
      '[data-session-window]',
    );
    if (sessionWindow)
      sessionWindow.style.minHeight = `${sessionWindow.getBoundingClientRect().height}px`;
    this.playback?.restart();
    this.render();
    this.schedule();
  }

  private togglePlayback() {
    if (this.playback?.view.isPlaying) this.pause();
    else this.resume();
  }

  private resume() {
    this.playback?.resume();
    this.render();
    this.schedule();
  }

  private pause() {
    if (!this.playback?.view.isPlaying) return;
    this.playback.pause();
    this.render();
    this.schedule();
  }

  private showComplete(reveal = false) {
    this.playback?.complete(reveal);
    this.render();
    this.schedule();
  }

  private render() {
    if (!this.panel || !this.playback) return;
    const entries = this.entries.get(this.panel) ?? [];
    const { frame, isPromptOnly, isComplete, isPlaying, isPaused, isPreview } =
      this.playback.view;
    if (isComplete) {
      const sessionWindow = this.querySelector<HTMLElement>(
        '[data-session-window]',
      );
      if (sessionWindow) sessionWindow.style.minHeight = '';
    }
    entries.forEach((entry, index) => {
      const isPast = index < frame.index || isComplete;
      entry.element.hidden = isPromptOnly || index > frame.index;
      const visibleCharacters = isPast
        ? entry.characters.length
        : index === frame.index
          ? frame.characters
          : 0;
      const text = entry.characters.slice(0, visibleCharacters).join('');
      if (entry.note.textContent !== text) entry.note.textContent = text;
      entry.note.toggleAttribute(
        'data-typing',
        index === frame.index && frame.phase === 'typing' && isPlaying,
      );
      entry.tool.hidden =
        !isPast && (index !== frame.index || frame.phase === 'typing');
      entry.lines.forEach((line, lineIndex) => {
        line.hidden = !isPast && lineIndex >= frame.lines;
      });
      const state = entry.tool.querySelector<HTMLElement>('[data-tool-state]');
      if (state) {
        const isRunning = !isPast && frame.phase === 'running';
        state.toggleAttribute('data-running', isRunning);
        state.textContent = isRunning
          ? 'running'
          : entry.tool.dataset.kind === 'FAIL'
            ? 'failed'
            : 'done';
      }
    });
    this.toggleAttribute('data-preview', isPreview);
    const overlay = this.querySelector<HTMLElement>('[data-preview-overlay]');
    if (overlay) {
      overlay.hidden = !isPreview && !isPaused;
      overlay.toggleAttribute('data-paused', isPaused);
    }
    const start = this.querySelector<HTMLButtonElement>('[data-start]');
    if (start) {
      start.setAttribute(
        'aria-label',
        isPaused ? 'Resume example session' : 'Play example session',
      );
      const label = start.querySelector('[data-start-label]');
      if (label) label.textContent = isPaused ? 'Resume' : 'Play example';
      const hint = start.querySelector<HTMLElement>('[data-resume-hint]');
      if (hint) hint.hidden = !isPaused;
    }
    // Move focus out before the overlay makes transcript controls unreachable.
    const covered = isPreview || isPaused;
    if (
      covered &&
      Array.from(this.entries.keys()).some((panel) =>
        panel.contains(document.activeElement),
      )
    ) {
      start?.focus({ preventScroll: true });
    }
    for (const panel of this.entries.keys()) panel.inert = covered;
    this.toggleAttribute('data-playing', isPlaying);
    this.toggleAttribute('data-complete', isComplete);
    const play = this.querySelector<HTMLButtonElement>('[data-play]');
    if (play) {
      const label = play.querySelector('[data-play-label]');
      if (label)
        label.textContent = isPlaying ? 'Pause' : isPaused ? 'Resume' : 'Play';
    }
    const finish = this.querySelector<HTMLButtonElement>('[data-finish]');
    if (finish) finish.disabled = isComplete && !isPreview;
  }

  private schedule() {
    clearTimeout(this.timer);
    if (!this.playback?.view.isPlaying || !this.isVisible || document.hidden)
      return;
    this.lastTick = performance.now();
    this.timer = setTimeout(() => {
      this.playback?.advance(performance.now() - this.lastTick);
      this.render();
      this.schedule();
    }, 32);
  }
}

if (!customElements.get('agent-demo'))
  customElements.define('agent-demo', AgentDemo);
