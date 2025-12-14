declare const ePub: any;

const book = ePub("/libros/libro.epub")

var rendition = book.renderTo("viewer", {
    width: "100%",
    height: 600,
    spread: "always"
});

rendition.display();
