<h1>ePub Reader</h1>
<script>
  import { SvelteReader } from "svelte-reader";
  export let data;
</script>
<div style="height: 100vh">
  <SvelteReader url="/books/{data.slug}.epub" title="eBook Reader"/>
</div>
