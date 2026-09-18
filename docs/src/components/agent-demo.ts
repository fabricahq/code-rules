/** @fileoverview Plays the homepage's tabbed agent transcripts; pauses for readers, hidden pages, and reduced-motion preferences. */
import { transcriptFrame } from './transcript-playback';

/** Cached, server-rendered content avoids replacing highlighted markup while the transcript streams. */
type Entry = {
  element: HTMLElement;
  note: HTMLElement;
  characters: string[];
  tool: HTMLElement;
  lines: HTMLElement[];
};

/** Owns one visible session and its playback clock. It never runs the illustrated commands. */
class AgentDemo extends HTMLElement {
  private entries = new Map<HTMLElement, Entry[]>();
  private panel: HTMLElement | undefined;
  private elapsed = 0;
  private isPlaying = true;
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
          event.target.closest('[data-transcript]')
        )
          this.pause();
      },
      { signal },
    );
    // Reader-initiated scrolling stops both streaming and automatic scroll following.
    this.addEventListener('wheel', () => this.pause(), {
      signal,
      passive: true,
    });
    this.addEventListener(
      'touchstart',
      (event) => {
        if (
          event.target instanceof Element &&
          event.target.closest('[data-transcript]')
        )
          this.pause();
      },
      { signal, passive: true },
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
    this.restart();
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
    if (!button) return;
    if (button.hasAttribute('data-tab')) this.selectTab(button);
    else if (button.hasAttribute('data-replay')) this.restart();
    else if (button.hasAttribute('data-finish')) this.showComplete();
    else if (button.hasAttribute('data-play')) {
      this.isPlaying = !this.isPlaying;
      this.render();
      this.schedule();
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
    this.restart();
  }

  private restart() {
    this.elapsed = this.motion.matches ? Infinity : 0;
    this.isPlaying = !this.motion.matches;
    this.render();
    const transcript =
      this.panel?.querySelector<HTMLElement>('[data-transcript]');
    if (transcript) transcript.scrollTop = 0;
    this.schedule();
  }

  private pause() {
    if (!this.isPlaying) return;
    this.isPlaying = false;
    this.render();
    this.schedule();
  }

  private showComplete() {
    this.elapsed = Infinity;
    this.isPlaying = false;
    this.render();
    this.schedule();
  }

  private render() {
    if (!this.panel) return;
    const entries = this.entries.get(this.panel) ?? [];
    const frame = transcriptFrame(
      entries.map((entry) => ({
        characters: entry.characters.length,
        lines: entry.lines.length,
      })),
      this.elapsed,
    );
    const isComplete = frame.phase === 'complete';
    if (isComplete) this.isPlaying = false;
    entries.forEach((entry, index) => {
      const isPast = index < frame.index || isComplete;
      entry.element.hidden = index > frame.index;
      const visibleCharacters = isPast
        ? entry.characters.length
        : index === frame.index
          ? frame.characters
          : 0;
      const text = entry.characters.slice(0, visibleCharacters).join('');
      if (entry.note.textContent !== text) entry.note.textContent = text;
      entry.note.toggleAttribute(
        'data-typing',
        index === frame.index && frame.phase === 'typing' && this.isPlaying,
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
    this.toggleAttribute('data-playing', this.isPlaying);
    this.toggleAttribute('data-complete', isComplete);
    const status = this.querySelector<HTMLElement>('[data-status]');
    if (status)
      status.textContent = isComplete
        ? 'Session complete'
        : !this.isPlaying
          ? 'Paused · scroll to read'
          : (entries[frame.index]?.element.dataset.title ?? 'Working');
    const play = this.querySelector<HTMLButtonElement>('[data-play]');
    if (play) {
      play.textContent = this.isPlaying ? 'Pause' : 'Play';
      play.disabled = isComplete;
    }
    const finish = this.querySelector<HTMLButtonElement>('[data-finish]');
    if (finish) finish.disabled = isComplete;
  }

  private schedule() {
    clearTimeout(this.timer);
    if (!this.isPlaying || !this.isVisible || document.hidden) return;
    this.lastTick = performance.now();
    this.timer = setTimeout(() => {
      this.elapsed += performance.now() - this.lastTick;
      this.render();
      const transcript =
        this.panel?.querySelector<HTMLElement>('[data-transcript]');
      if (transcript) transcript.scrollTop = transcript.scrollHeight;
      this.schedule();
    }, 32);
  }
}

if (!customElements.get('agent-demo'))
  customElements.define('agent-demo', AgentDemo);
