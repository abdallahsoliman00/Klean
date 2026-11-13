/* The API the rest of the UI imports. `npm run dev:mock` (Vite mode "mock") swaps in the in-memory mock;
   every other build talks to the backend over HTTP. The mode is compared with a literal on purpose:
   the bundler then drops the mock from production builds. */
import type { Api } from '../types';
import { httpApi } from './http';
import { mockApi } from './mock';

export const api: Api = import.meta.env.MODE === 'mock' ? mockApi : httpApi;
export { ApiError } from './errors';
