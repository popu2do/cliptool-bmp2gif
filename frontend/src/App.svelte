<script>
  import { onMount } from 'svelte';
  import { flip } from 'svelte/animate';
  import { dndzone } from 'svelte-dnd-action';
  import { Check, Copy, Pin, PinOff, RotateCcw, Shuffle, Trash2, X } from '@lucide/svelte';
  import {
    AddDroppedFiles,
    ClearFrames,
    GenerateGIF,
    GetFrames,
    RemoveFrame,
    ReorderFrames,
    ScanClipboard,
    SetAlwaysOnTop,
  } from '../wailsjs/go/main/App';
  import { OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime';
  import { frameIDs, normalizeFrames } from './frameModel';

  const scanIntervalMS = 700;
  const flipDurationMS = 120;

  let frames = [];
  let selectedID = '';
  let delayMS = 500;
  let alwaysOnTop = false;
  let generating = false;
  let dragging = false;
  let status = '监听中';
  let statusError = false;
  let scanTimer;

  function isGenericClipboardError(message) {
    return message.startsWith('剪贴板里没有文件列表');
  }

  function canGenerate() {
    return frames.length >= 2 && !generating;
  }

  async function refreshFrames() {
    frames = normalizeFrames(await GetFrames());
    if (!frames.some((frame) => frame.id === selectedID)) {
      selectedID = '';
    }
  }

  async function scanClipboard() {
    if (generating || dragging) return;

    try {
      const result = await ScanClipboard();
      frames = normalizeFrames(result.Frames);
      if (result.Message && result.Message !== '监听中') {
        if (statusError && result.Error && isGenericClipboardError(result.Message)) {
          return;
        }
        status = result.Message;
        statusError = Boolean(result.Error);
      }
      if (!frames.some((frame) => frame.id === selectedID)) {
        selectedID = '';
      }
    } catch (error) {
      status = `读取剪贴板失败：${String(error)}`;
      statusError = true;
    }
  }

  async function removeSelected() {
    if (!selectedID || generating) return;
    frames = normalizeFrames(await RemoveFrame(selectedID));
    selectedID = '';
    status = '已删除选中图片';
    statusError = false;
  }

  async function clearAll() {
    if (generating) return;
    await ClearFrames();
    frames = [];
    selectedID = '';
    status = '已清空';
    statusError = false;
  }

  async function reorderByIDs(ids) {
    frames = normalizeFrames(await ReorderFrames(ids));
  }

  async function sortNatural() {
    const nextFrames = [...frames].sort((a, b) =>
      a.Name.localeCompare(b.Name, 'zh-Hans-CN', { numeric: true, sensitivity: 'base' }),
    );
    await reorderByIDs(frameIDs(nextFrames));
    status = '已按文件名排序';
    statusError = false;
  }

  async function reverseOrder() {
    await reorderByIDs(frameIDs([...frames].reverse()));
    status = '已反转顺序';
    statusError = false;
  }

  async function generateGIF() {
    if (!canGenerate()) return;
    generating = true;
    status = '生成中';
    statusError = false;

    try {
      const result = await GenerateGIF({ DelayMS: Number(delayMS) });
      status = result.Message;
      statusError = !result.OK;
      if (result.OK) {
        frames = [];
        selectedID = '';
      } else {
        await refreshFrames();
      }
    } catch (error) {
      status = String(error);
      statusError = true;
      await refreshFrames();
    } finally {
      generating = false;
    }
  }

  async function toggleAlwaysOnTop() {
    alwaysOnTop = !alwaysOnTop;
    await SetAlwaysOnTop(alwaysOnTop);
    status = alwaysOnTop ? '窗口已置顶' : '窗口取消置顶';
    statusError = false;
  }

  function handleConsider(event) {
    dragging = true;
    frames = normalizeFrames(event.detail.items);
  }

  async function handleFinalize(event) {
    const nextFrames = normalizeFrames(event.detail.items);
    frames = nextFrames;
    try {
      await reorderByIDs(frameIDs(nextFrames));
    } finally {
      dragging = false;
    }
  }

  function isExternalFileDrag(event) {
    return Array.from(event.dataTransfer?.types || []).includes('Files');
  }

  function preventExternalFileDrop(event) {
    if (!isExternalFileDrag(event)) return;
    event.preventDefault();
    event.stopPropagation();
  }

  async function addDroppedFiles(paths) {
    if (generating || !paths?.length) return;
    dragging = false;
    try {
      const result = await AddDroppedFiles(paths);
      frames = normalizeFrames(result.Frames);
      status = result.Message;
      statusError = Boolean(result.Error);
      if (!frames.some((frame) => frame.id === selectedID)) {
        selectedID = '';
      }
    } catch (error) {
      status = `拖拽导入失败：${String(error)}`;
      statusError = true;
      await refreshFrames();
    }
  }

  function handleKeydown(event) {
    if (event.key === 'Enter') {
      event.preventDefault();
      generateGIF();
    }
    if (event.key === 'Delete') {
      event.preventDefault();
      removeSelected();
    }
  }

  onMount(() => {
    refreshFrames();
    SetAlwaysOnTop(alwaysOnTop);
    scanTimer = window.setInterval(scanClipboard, scanIntervalMS);
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('dragover', preventExternalFileDrop);
    window.addEventListener('drop', preventExternalFileDrop);
    OnFileDrop((_x, _y, paths) => {
      addDroppedFiles(paths);
    }, false);
    return () => {
      window.clearInterval(scanTimer);
      window.removeEventListener('keydown', handleKeydown);
      window.removeEventListener('dragover', preventExternalFileDrop);
      window.removeEventListener('drop', preventExternalFileDrop);
      OnFileDropOff();
    };
  });
</script>

<main class="app-shell">
  <header class="toolbar">
    <div class="title-area">
      <Copy size={18} />
      <h1>ClipTool</h1>
      <span class="frame-count">{frames.length} 帧</span>
    </div>

    <button
      class:active={alwaysOnTop}
      class="icon-button"
      title={alwaysOnTop ? '取消置顶' : '窗口置顶'}
      type="button"
      on:click={toggleAlwaysOnTop}
    >
      {#if alwaysOnTop}
        <Pin size={17} />
      {:else}
        <PinOff size={17} />
      {/if}
    </button>
  </header>

  <section class="frame-board" aria-label="图片帧列表">
    {#if frames.length === 0}
      <div class="empty-state">
        <Copy size={26} />
        <span>复制图片后自动追加</span>
      </div>
    {:else}
      <div
        class="frame-grid"
        use:dndzone={{ items: frames, flipDurationMs: flipDurationMS }}
        on:consider={handleConsider}
        on:finalize={handleFinalize}
      >
        {#each frames as frame, index (frame.id)}
          <button
            animate:flip={{ duration: flipDurationMS }}
            class:selected={selectedID === frame.id}
            class="frame-tile"
            title={frame.Path}
            type="button"
            on:click={() => (selectedID = frame.id)}
          >
            <span class="frame-index">{index + 1}</span>
            <img src={frame.ThumbDataURL} alt={frame.Name} />
            <span class="frame-name">{frame.Name}</span>
            <span class="frame-meta">{frame.Width}x{frame.Height} · {frame.Format}</span>
          </button>
        {/each}
      </div>
    {/if}
  </section>

  <footer class="control-panel">
    <div class="delay-control">
      <label for="delay">Gif图片间隔时间</label>
      <input id="delay" min="100" max="3000" step="50" type="range" bind:value={delayMS} />
      <input min="100" max="3000" step="50" type="number" bind:value={delayMS} />
      <span>ms</span>
    </div>

    <div class="actions">
      <button title="按文件名自然排序" type="button" disabled={frames.length < 2 || generating} on:click={sortNatural}>
        <Shuffle size={16} />
        <span>自然</span>
      </button>
      <button title="反转当前顺序" type="button" disabled={frames.length < 2 || generating} on:click={reverseOrder}>
        <RotateCcw size={16} />
        <span>反转</span>
      </button>
      <button title="删除选中图片" type="button" disabled={!selectedID || generating} on:click={removeSelected}>
        <Trash2 size={16} />
        <span>删除</span>
      </button>
      <button title="清空当前列表" type="button" disabled={frames.length === 0 || generating} on:click={clearAll}>
        <X size={16} />
        <span>清空</span>
      </button>
      <button class="primary" title="生成 GIF 并复制到剪贴板，也可按 Enter" type="button" disabled={!canGenerate()} on:click={generateGIF}>
        <Check size={17} />
        <span>{generating ? '生成中' : '生成 GIF / Enter'}</span>
      </button>
    </div>
  </footer>

  <div class:error={statusError} class="status-bar">{status}</div>
</main>
