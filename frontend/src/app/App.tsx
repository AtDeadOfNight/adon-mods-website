import { Layout } from '@/app/Layout';
import { RouteRecord } from 'vite-react-ssg';
import { HomePage } from '../pages';
import { PurchasePage } from '@/pages/purchase';
import { NotFoundPage } from '@/pages/not-found';

export const routes: RouteRecord[] = [
	{
		path: '/',
		element: <Layout />,
		children: [
			{ index: true, element: <HomePage /> },
			{
				path: 'purchase',
				element: <PurchasePage />
			},
			{
				path: '*',
				element: <NotFoundPage />
			}
		]
	}
];
