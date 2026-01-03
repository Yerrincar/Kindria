<script lang="ts">
  import { onMount } from 'svelte';
  interface Book {
    file_name: string;
    bookpath: string;
    title: string;
    author: string;
  }

  let books: Book[] = [];

  onMount(async function () {
    const endpointBooks = '/api/books/getbooks';
    const response = await fetch(endpointBooks);
    books = await response.json();
  });
</script>

<!--- Not Script Part --->

<h1>Kindria</h1>
<div class="library">
  {#each books as book}
    <div class="book-card">
      <a href="/reader/{encodeURIComponent(book.file_name)}">
        <img src="/covers/{book.title}.jpg" alt="" />
        <p>{book.title}</p>
      </a>
    </div>
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
