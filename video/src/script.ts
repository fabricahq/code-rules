/** @fileoverview The tour's scenes and narration, in order; the voiceover script and the composition both read it. */

/** One scene of the tour and the narration spoken over it. */
export type SceneScript = {
  /** Stable ID, used for the scene's component and its voiceover file name. */
  id: SceneId;
  narration: string;
};

/** The IDs of the tour's scenes. */
export type SceneId =
  | 'the-mess'
  | 'what-a-library-is'
  | 'the-ask'
  | 'add-their-library'
  | 'sync'
  | 'rules-too'
  | 'skills-and-rules'
  | 'agent-at-work'
  | 'capture-what-you-learn'
  | 'close';

/** The tour's scenes, in the order they play. */
export const SCENES = [
  {
    id: 'the-mess',
    narration: 'Your agents already use skills. But where did each one come from, and which version are they running?',
  },
  {
    id: 'what-a-library-is',
    narration:
      'With Syllabus, teams and organizations share their skills in a Syllabus library. It’s an ordinary Git repository of skills, but it gives each skill its own version, change notes with every release, and a recommendation level that tells consumers whether each skill is required, recommended, or just available. A library can also publish rules, which we’ll get to shortly.',
  },
  {
    id: 'the-ask',
    narration:
      'Now, let’s say you like the way the payments team writes automated tests, and you want your agent to write them the same way.',
  },
  {
    id: 'add-their-library',
    narration: 'The payments team publishes a Syllabus library, so add it as a source and include their testing skill.',
  },
  {
    id: 'sync',
    narration:
      'Running syllabus project sync installs the skill into every folder your agents read, and records exactly where it came from.',
  },
  {
    id: 'rules-too',
    narration:
      'But the payments team has more than a skill for writing good tests. They also have a set of rules for guaranteeing that a critical operation actually happened. So let’s add those rules too.',
  },
  {
    id: 'skills-and-rules',
    narration:
      'Rules are a more granular kind of guidance than skills: one best practice each, with its own version. Syllabus lets you pull in rules, skills, or both, from many different sources, or write your own.',
  },
  {
    id: 'agent-at-work',
    narration:
      'Now when your agent writes tests, it follows the payments team’s skill and checks its work against their rules.',
  },
  {
    id: 'capture-what-you-learn',
    narration:
      'And every time you have an insight, record it as a new rule, a new skill, or an update to one. Share it with others in a library, or keep it in this project.',
  },
  {
    id: 'close',
    narration: 'Syllabus. The package manager for your agent skills and rules.',
  },
] as const satisfies ReadonlyArray<SceneScript>;
