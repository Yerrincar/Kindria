<script lang="ts">
  import { onMount } from 'svelte';
  interface Book {
    book_name: string;
    cover_path: string;
    metadata: {
      title: string;
      author: string;
    };
  }

  let books: Book[] = [];

  onMount(async function () {
    const endpointBooks = '/api/getbooks';
    const response = await fetch(endpointBooks);
    books = await response.json();
  });
</script>

<!--- Not Script Part --->

<h1>Kindria</h1>
<div class="library">
  {#each books as book}
    <div class="book-card">
      <a href="/reader/{encodeURIComponent(book.book_name)}">
        <img
          src="/api/getCovers?book={book.book_name}&path={book.cover_path}"
          alt=""
        />
        <p>{book.metadata.title}</p>
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
