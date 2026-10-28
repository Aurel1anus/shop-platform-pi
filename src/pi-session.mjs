import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  createAgentSession,
  DefaultResourceLoader,
  ModelRuntime,
  SessionManager,
  SettingsManager,
} from '@earendil-works/pi-coding-agent';

export const projectRoot = dirname(dirname(fileURLToPath(import.meta.url)));
export const runtimeDir = join(projectRoot, '.pi', 'runtime');

export async function createShopSession({ modelRuntime } = {}) {
  const settings = JSON.parse(await readFile(join(projectRoot, '.pi', 'settings.json'), 'utf8'));
  const settingsManager = SettingsManager.inMemory(settings, { projectTrusted: true });
  const resourceLoader = new DefaultResourceLoader({
    cwd: projectRoot,
    agentDir: runtimeDir,
    settingsManager,
    // 禁止自动发现全局、祖先或额外配置资源，只加载下面明确列出的项目资源。
    noExtensions: true,
    noSkills: true,
    noPromptTemplates: true,
    additionalExtensionPaths: [
      join(projectRoot, '.pi'),
    ],
    additionalSkillPaths: [join(projectRoot, '.pi/skills')],
    noContextFiles: true,
    noThemes: true,
  });
  await resourceLoader.reload();
  const extensions = resourceLoader.getExtensions();
  if (extensions.errors.length) {
    throw new Error(extensions.errors.map((item) => item.error).join('\n'));
  }

  // The SDK loads only this project's resources; model credentials stay in the user's Pi config.
  const models = modelRuntime ?? await ModelRuntime.create();
  const provider = process.env.PI_PROVIDER ?? settings.defaultProvider;
  const modelId = process.env.PI_MODEL ?? settings.defaultModel;
  const model = models.getModel(provider, modelId);
  if (!model) throw new Error(`找不到模型 ${provider}/${modelId}，请检查 pi 模型配置。`);

  const result = await createAgentSession({
    cwd: projectRoot,
    agentDir: runtimeDir,
    modelRuntime: models,
    model,
    resourceLoader,
    settingsManager,
    sessionManager: SessionManager.inMemory(projectRoot),
    tools: settings.defaultTools,
  });
  await result.session.bindExtensions({ mode: 'print' });
  return { ...result, resourceLoader };
}
