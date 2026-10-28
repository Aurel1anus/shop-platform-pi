import assert from 'node:assert/strict';
import { realpath } from 'node:fs/promises';
import { join } from 'node:path';
import { ModelRuntime } from '@earendil-works/pi-coding-agent';
import { createShopSession, projectRoot, runtimeDir } from '../src/pi-session.mjs';

const checking = process.argv[2] === '--check';
const prompt = process.argv.slice(2).join(' ').trim();
if (!checking && !prompt) {
  console.error('用法：npm run chat -- "你好"；交互模式：npm run pi；离线检查：npm run check');
  process.exit(1);
}

let session;
let timer;
try {
  const modelRuntime = checking ? await ModelRuntime.create({
    authPath: join(runtimeDir, 'check-auth.json'),
    modelsPath: null,
    refreshOnCreate: false,
  }) : undefined;
  const created = await createShopSession({ modelRuntime });
  session = created.session;
  if (checking) {
    assert.deepEqual(session.getActiveToolNames().sort(), ['get_product', 'list_product_skus', 'list_products', 'read', 'shop_status']);
    const { skills, diagnostics } = created.resourceLoader.getSkills();
    assert.deepEqual(skills.map((skill) => skill.name).sort(), ['knowledge', 'orders', 'shopping']);
    assert.equal(diagnostics.length, 0, JSON.stringify(diagnostics));
    const extensions = created.resourceLoader.getExtensions().extensions;
    assert.deepEqual(
      (await Promise.all(extensions.map((extension) => realpath(extension.resolvedPath)))).sort(),
      (await Promise.all(['status.ts', 'commerce/index.ts', 'harness/index.ts'].map(
        (path) => realpath(join(projectRoot, '.pi/extensions', path)),
      ))).sort(),
    );
    assert.deepEqual(
      (await Promise.all(skills.map((skill) => realpath(skill.filePath)))).sort(),
      (await Promise.all(['knowledge', 'orders', 'shopping'].map(
        (name) => realpath(join(projectRoot, '.pi/skills', name, 'SKILL.md')),
      ))).sort(),
    );
    assert.deepEqual(created.resourceLoader.getAgentsFiles().agentsFiles, []);
    assert.deepEqual(created.resourceLoader.getPrompts().prompts, []);
    const tool = session.agent.state.tools.find((item) => item.name === 'shop_status');
    const result = await tool.execute('pi-check', {});
    assert.equal(result.details.backendConnection, 'not_checked');
    assert.equal(result.details.commerce, 'product_read_only');
    assert.deepEqual(result.details.commerceTools, ['get_product', 'list_products', 'list_product_skus']);
    assert.equal(result.details.memory, 'placeholder');
    assert.equal(result.details.rag, 'placeholder');
    console.log('Pi SDK initialized; 3 extensions and 3 skills loaded; shop_status executed.');
    console.log('Resource paths verified: project allowlist only; no context files or prompt templates loaded.');
    console.log(`Model: ${session.model.provider}/${session.model.id}; no model API request was made.`);
  } else {
    let failure;
    session.subscribe((event) => {
      if (event.type === 'message_update' && event.assistantMessageEvent.type === 'text_delta') {
        process.stdout.write(event.assistantMessageEvent.delta);
      }
      if (event.type === 'tool_execution_start') console.error(`[tool] ${event.toolName}`);
      if (event.type === 'message_end' && event.message.role === 'assistant' &&
          ['error', 'aborted'].includes(event.message.stopReason)) {
        failure = event.message.errorMessage ?? event.message.stopReason;
      }
    });
    timer = setTimeout(() => { void session.abort(); }, 60_000);
    await session.prompt(prompt);
    if (failure) throw new Error(failure);
    console.log();
  }
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
} finally {
  clearTimeout(timer);
  session?.dispose();
}
