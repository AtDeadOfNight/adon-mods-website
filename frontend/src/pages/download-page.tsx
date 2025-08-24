import { useParams } from 'react-router-dom';
import React from 'react';

export function DownloadPage() {
	const [isRu, setIsRu] = React.useState(false);

	React.useEffect(() => {
		setIsRu(navigator.language.startsWith('ru'));
	}, []);

	const file = useParams().file;

	const downloadUrl = file
		? `/download?file=${encodeURIComponent(file)}`
		: null;

	return (
		<div className="flex items-center justify-center text-white">
			{downloadUrl ? (
				<a
					href={downloadUrl}
					className="text-white underline underline-offset-2"
				>
					{isRu ? 'Нажмите, чтобы скачать' : 'Click here to download'}
				</a>
			) : isRu ? (
				'Файл не найден'
			) : (
				'File not found'
			)}
		</div>
	);
}
