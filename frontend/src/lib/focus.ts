/* Elements other parts of the UI hand focus to. */
import { createRef } from 'react';

/** The composer's textarea: where focus goes back to by default. */
export const composerInput = createRef<HTMLTextAreaElement>();
/** The artifact switcher in the panel header, focused when a card opens the panel. */
export const artifactSwitch = createRef<HTMLButtonElement>();
/** The system prompt field, focused when the chat settings panel opens. */
export const systemPromptInput = createRef<HTMLTextAreaElement>();

export const focusComposer = () => composerInput.current?.focus();
