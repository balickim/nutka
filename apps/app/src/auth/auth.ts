// Exposes the learner session facade backed by the learner-specific cookie session API.

import { createAuthRealm, classifyAuthError as classifyRealmError } from "./realm";

export type Learner = {
  id: string;
  email: string;
  name?: string;
  display_name?: string;
  [key: string]: unknown;
};
export type MeResult = Awaited<ReturnType<typeof learnerAuth.fetchMe>>;
export type AuthState = ReturnType<typeof learnerAuth.getState>;

export const learnerAuth = createAuthRealm<Learner>({
  collection: "learners",
  mePath: "/api/learners/auth/me",
  logoutPath: "/api/learners/auth/logout",
  channelName: "nutka-learner-auth",
  displayName: (record) => record.name || record.display_name || record.email,
});

export const subscribe = learnerAuth.subscribe;
export const getAuthState = learnerAuth.getState;
export const getLearnerDisplayName = learnerAuth.getDisplayName;
export const fetchAuthMe = learnerAuth.fetchMe;
export const bootstrapAuth = learnerAuth.bootstrap;
export const login = learnerAuth.login;
export const logout = learnerAuth.logout;
export const classifyAuthError = classifyRealmError;
export type AuthErrorKind = ReturnType<typeof classifyAuthError>;
