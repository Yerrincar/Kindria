<script>
    import { onMount } from "svelte";	
    
    let books = [];
    
    onMount(async function () {
	const endpointBooks = "/api/getbooks"
    	const response = await fetch(endpointBooks);
    	books = await response.json();
    });
</script>

<!--- Not Script Part --->

<h1>Kindria</h1>
<div class='library'>
{#each books as book}
    <p>{book.metadata.title}</p>
    <p>{book.metadata.author}</p>
    <img src="/api/getCovers?book={book.book_name}&path={book.cover_path}" alt={book.metadata.title} />
{/each} 
</div>
<style>
    .library {
        width: 100%;
        display: grid;
        grid-template-columns: repeat(5, 1fr);
        grid-gap: 8px;
    }
    figure,
    img {
        width: 100%;
        margin: 0;
    }
</style> 
