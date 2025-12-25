<script>
    import { onMount } from "svelte";	
    const endpoint = "/api/getbooks"
    let books = []
    onMount(async function () {
    	const response = await fetch(endpoint);
    	books = await response.json();
    });
</script>
<h1>Kindria</h1>
{#each books as book}
    <p>{book.metadata.title}</p>
    <p>{book.metadata.author}</p>
{/each}
	


