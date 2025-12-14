import * as pdfjsLib from 'pdfjs-dist';

pdfjsLib.GlobalWorkerOptions.workerSrc = '/worker/pdf.worker.min.mjs';

async function readPDF() {
    try {
        const pdf = await pdfjsLib.getDocument("/libros/libro.pdf").promise;

        const page = await pdf.getPage(1);
        const scale = 1.5;
        const viewport = page.getViewport({ scale: scale });

        const canvas = document.createElement('canvas');
        const context = canvas.getContext('2d');
        canvas.height = viewport.height;
        canvas.width = viewport.width;

        if (!context) {
            console.error("Could not obtain 2D rendering context from canvas");
            return;
        }

        const renderContext = {
            canvasContext: context,
            canvas: canvas,
            viewport: viewport
        };
        await page.render(renderContext).promise;
        document.body.appendChild(canvas);
        pdf.destroy();
    }
    catch (error) {
        console.error("Error rendering the PDF: ", error);
    }
}


window.addEventListener('load', readPDF)
