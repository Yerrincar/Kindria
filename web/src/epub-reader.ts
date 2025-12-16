let current = 0;

function loadChapter(n: number): void {
    fetch(`/chapter?n=${n}`)
        .then(res => res.text())
        .then(html => {
            const iframe = document.getElementById("viewer") as HTMLIFrameElement;
            iframe.srcdoc = html;
            current = n;
        });
}

function nextChapter(): void {
    loadChapter(current + 1);
}

function toggleNight(): void {
    document.body.classList.toggle("night");
}

function setFontSize(size: string): void {
    const iframe = document.getElementById("viewer") as HTMLIFrameElement;
    const doc = iframe.contentDocument || iframe.contentWindow?.document;
    if (doc && doc.body) {
        doc.body.style.fontSize = `${size}px`;
    }
}

window.onload = () => loadChapter(0);

// Export for global access (optional)
(window as any).toggleNight = toggleNight;
(window as any).nextChapter = nextChapter;
(window as any).setFontSize = setFontSize;


