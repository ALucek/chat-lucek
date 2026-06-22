'use client';

import { useAuth } from '@/lib/auth-context';

export default function Home() {
  return (
    <div className="flex h-full items-center justify-center">
      <p className="text-sm text-gray-500">Select or create a conversation</p>
    </div>
  );
}
