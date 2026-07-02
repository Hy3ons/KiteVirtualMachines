import { readDebugState, writeDebugState } from './debugMockState';
import { debugVmApi } from './debugMockVm';
import type {
  AdminCreateVmOfferPayload,
  ClaimVmOfferPayload,
  VmOffer,
  VmOfferResponse,
  VmOffersResponse,
  VmResponse,
} from './types';

const resolveTargetNamespace = (payload: AdminCreateVmOfferPayload): string => {
  const state = readDebugState();
  const targetUser = payload.targetUser?.trim();
  if (!targetUser) {
    return payload.targetNamespace?.trim() || 'hyeonseok-ns';
  }

  const user = state.users.find((candidate) => candidate.username === targetUser);
  return user?.namespace || targetUser;
};

const activeOffers = (offers: readonly VmOffer[]): readonly VmOffer[] =>
  offers.filter((offer) => offer.phase === 'Available' && new Date(offer.expiresAt).getTime() > Date.now());

export const debugOfferApi = {
  getOffers: async (): Promise<VmOffersResponse> => ({ offers: activeOffers(readDebugState().offers) }),

  claimOffer: async (name: string, payload: ClaimVmOfferPayload): Promise<VmResponse> => {
    const state = readDebugState();
    const offer = state.offers.find((candidate) => candidate.name === name && candidate.phase === 'Available');
    if (!offer) {
      throw new Error('VM offer was not found');
    }

    const claimed: VmOffer = {
      ...offer,
      phase: 'Claimed',
      claimedBy: 'debug',
      message: 'Debug offer claimed',
    };
    writeDebugState({
      ...state,
      offers: state.offers.map((candidate) => (candidate.name === name ? claimed : candidate)),
    });

    const response = await debugVmApi.createVm({
      name: payload.vmName,
      domainPrefix: payload.domainPrefix,
      sshId: payload.sshId,
      sshPassword: payload.initialLoginPassword,
      cpu: offer.cpu,
      memory: offer.memory,
      disk: offer.disk,
      powerState: payload.powerState || 'On',
    });
    const nextState = readDebugState();
    writeDebugState({
      ...nextState,
      offers: nextState.offers.filter((candidate) => candidate.name !== name),
    });
    return response;
  },

  createVmOffer: async (payload: AdminCreateVmOfferPayload): Promise<VmOfferResponse> => {
    const state = readDebugState();
    const namespace = resolveTargetNamespace(payload);
    const name = `offer-${state.nextOfferId}`;
    const offer: VmOffer = {
      id: `${namespace}/${name}`,
      name,
      namespace,
      cpu: payload.cpu,
      memory: payload.memory,
      disk: payload.disk,
      image: payload.image || 'ubuntu-22.04',
      expiresAt: payload.expiresAt || new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
      createdBy: 'admin',
      phase: 'Available',
      claimedBy: '',
      message: 'Debug offer assigned by admin',
    };

    writeDebugState({
      ...state,
      nextOfferId: state.nextOfferId + 1,
      offers: [...state.offers, offer],
    });
    return { offer };
  },

  deleteVmOffer: async (namespace: string, name: string): Promise<{ readonly message: string }> => {
    const state = readDebugState();
    writeDebugState({
      ...state,
      offers: state.offers.filter((offer) => offer.namespace !== namespace || offer.name !== name),
    });
    return { message: 'VM offer deleted' };
  },
};
