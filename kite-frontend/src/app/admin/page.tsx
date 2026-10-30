import { redirect } from 'next/navigation';

// Admin keeps the administration entry URL pointed at its dashboard.
export default function Admin() { redirect('/admin/dashboard'); }
