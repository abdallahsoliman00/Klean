import type { Message } from '../../types';
import { AssistantMessage } from './AssistantMessage';
import { UserMessage } from './UserMessage';

export interface MessageProps {
  message: Message;
  /** The newest message: its tools stay visible, and a reply can be regenerated. */
  isLast: boolean;
  /** Animate in (the message just sent). */
  enter: boolean;
}

export function MessageView(props: MessageProps) {
  return props.message.role === 'user' ? <UserMessage {...props} /> : <AssistantMessage {...props} />;
}

export const messageClass = (role: string, isLast: boolean, enter: boolean) => `msg ${role}${isLast ? ' last' : ''}${enter ? ' enter' : ''}`;
