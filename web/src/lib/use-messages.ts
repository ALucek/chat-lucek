'use client';

import { useEffect, useState } from 'react';
import { type Message, getMessages, ApiError } from './api';

export interface UseMessages {
  messages: Message[];
  loading: boolean;
  error: string | null;
  notFound: boolean;
}

interface State extends UseMessages {
  id: number;
}

const LOADING: UseMessages = {
  messages: [],
  loading: true,
  error: null,
  notFound: false,
};

export function useMessages(id: number): UseMessages {
  const [state, setState] = useState<State>({ id, ...LOADING });

  useEffect(() => {
    let ignore = false;
    getMessages(id)
      .then((m) => {
        if (!ignore)
          setState({
            id,
            messages: m,
            loading: false,
            error: null,
            notFound: false,
          });
      })
      .catch((e) => {
        if (ignore) return;
        const notFound = e instanceof ApiError && e.status === 404;
        setState({
          id,
          messages: [],
          loading: false,
          error: notFound ? null : 'Couldn’t load messages',
          notFound,
        });
      });
    return () => {
      ignore = true;
    };
  }, [id]);

  // When id has changed but the effect hasn't resolved yet, state still
  // describes the previous id — report loading until the new fetch lands.
  if (state.id !== id) return { ...LOADING };
  return state;
}
