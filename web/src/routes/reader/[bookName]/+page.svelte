<script lang="ts">
  import { SvelteReader } from 'svelte-reader';
  import type { PageData } from './$types';

  export let data: PageData;
  let rendition: any;
  const getRendition = (val: any) => (rendition = val);
  const tocChanged = (toc: any) => console.log('Table of Contents:', toc);
</script>

<h1>Reading: {data.bookFile}</h1>

<div style="height: 100vh">
  <SvelteReader
    url="/books/{data.bookFile}"
    title="eBook Reader"
    {getRendition}
    {tocChanged}
  />
</div>
