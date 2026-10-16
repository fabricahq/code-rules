# Syllabus tour video

A narrated tour of Syllabus for the docs homepage, made with [Remotion](https://www.remotion.dev). A developer installs the payments team's testing skill from their Syllabus library, adds the team's rules, and records a new rule of their own.

The narration lives in [`src/script.ts`](src/script.ts). Each scene plays for as long as its narration clip, so changing a line retimes its scene.

## Preview and render

From this directory:

```sh
bun install
bun run studio           # preview in the browser
bun run render:silent    # out/syllabus-tour-silent.mp4, captions only
bun run render           # out/syllabus-tour.mp4, with the voiceover
```

Without a voiceover, scenes are timed from an estimated speaking rate and render silent with captions.

## Generate the voiceover

The voiceover comes from [ElevenLabs](https://elevenlabs.io). Set an API key and a voice ID, then run:

```sh
ELEVENLABS_API_KEY=... ELEVENLABS_VOICE_ID=... bun run voiceover
```

It writes one clip per scene to `public/voiceover/` and word timings for the captions to `public/voiceover/manifest.json`; commit both. It regenerates only the scenes whose narration, voice, or model changed. Set `ELEVENLABS_MODEL_ID` to use a model other than `eleven_multilingual_v2`.

## Check

```sh
bun run check
```
