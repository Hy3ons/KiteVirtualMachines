import { DEBUG_ADMIN_SESSION } from '../config/debug';
import { readDebugState, writeDebugState } from './debugMockState';
import type { AuthResponse, LoginCredentials, SignupPayload, SignupResponse } from './types';

export const debugAuthApi = {
  login: async (credentials: LoginCredentials): Promise<AuthResponse> => ({
    accessToken: DEBUG_ADMIN_SESSION.token,
    user: {
      access_level: DEBUG_ADMIN_SESSION.accessLevel,
      username: credentials.email === 'admin' ? 'admin' : credentials.email,
      namespace: DEBUG_ADMIN_SESSION.namespace,
      profile_image: DEBUG_ADMIN_SESSION.profileImage,
    },
  }),

  signup: async (payload: SignupPayload): Promise<SignupResponse> => {
    const state = readDebugState();
    const user = {
      username: payload.username,
      email: payload.email,
      namespace: `${payload.username}-ns`,
      accessLevel: 0,
      status: 'Pending',
    } satisfies SignupDebugUser;

    writeDebugState({
      ...state,
      users: [...state.users.filter((existing) => existing.username !== payload.username), user],
    });

    return {
      message: 'debug user created successfully',
      user: {
        access_level: user.accessLevel,
        username: user.username,
        namespace: user.namespace,
        profile_image: payload.profile_image,
        email: user.email,
      },
    };
  },

  getMe: async (): Promise<{ readonly user: AuthResponse['user'] }> => ({
    user: {
      access_level: DEBUG_ADMIN_SESSION.accessLevel,
      username: DEBUG_ADMIN_SESSION.username,
      namespace: DEBUG_ADMIN_SESSION.namespace,
      profile_image: DEBUG_ADMIN_SESSION.profileImage,
    },
  }),
};

type SignupDebugUser = {
  readonly username: string;
  readonly email: string;
  readonly namespace: string;
  readonly accessLevel: number;
  readonly status: 'Pending';
};
