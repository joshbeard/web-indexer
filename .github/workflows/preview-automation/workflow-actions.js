// Workflow action handlers for GitHub Actions
// Contains all the logic for workflow steps to minimize inline code

const { createStatusComment, updateStatusComment, createAutoCleanupComment, safeUpdateComment } = require('./comment-manager');

/**
 * Handle initial comment creation after command parsing
 */
async function handleInitialComment(github, context, core, stepOutputs) {
  const prNumber = stepOutputs.pr_number;
  const commentUser = stepOutputs.comment_user;
  const customArgs = stepOutputs.custom_args || '';
  const isCleanup = stepOutputs.cleanup_only === 'true';
  const command = isCleanup ? '/preview cleanup' : customArgs ? `/preview ${customArgs}` : '/preview';

  const commentId = await createStatusComment(github, context, prNumber, 'queuing', {
    command,
    commentUser,
    customArgs,
    isCleanup
  });

  core.setOutput('comment_id', commentId);
  return commentId;
}

/**
 * Handle pending approval comment update
 */
async function handlePendingApproval(github, context, commentId, stepOutputs) {
  const commentUser = stepOutputs.comment_user;
  const customArgs = stepOutputs.custom_args || '';
  const isCleanup = stepOutputs.cleanup_only === 'true';
  const command = isCleanup ? '/preview cleanup' : customArgs ? `/preview ${customArgs}` : '/preview';

  await updateStatusComment(github, context, commentId, 'pending', {
    command,
    commentUser,
    customArgs,
    isCleanup
  });
}

/**
 * Handle preview generation starting comment update
 */
async function handlePreviewStarting(github, context, env) {
  const commentId = env.STATUS_COMMENT_ID;
  const commentUser = env.COMMENT_USER;
  const customArgs = env.CUSTOM_ARGS || '';
  const command = customArgs ? `/preview ${customArgs}` : '/preview';
  const approver = env.GITHUB_ACTOR;
  const runUrl = `https://github.com/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}`;

  await updateStatusComment(github, context, commentId, 'running', {
    command,
    commentUser,
    approver,
    customArgs,
    runUrl
  });
}

/**
 * Handle preview generation success comment update
 */
async function handlePreviewSuccess(github, context, env) {
  const commentId = env.STATUS_COMMENT_ID;
  const commentUser = env.COMMENT_USER;
  const customArgs = env.CUSTOM_ARGS || '';
  const command = customArgs ? `/preview ${customArgs}` : '/preview';
  const approver = env.GITHUB_ACTOR;
  const runUrl = `https://github.com/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}`;
  const s3Url = env.S3_URL;
  const bucketName = env.BUCKET_NAME;

  await safeUpdateComment(github, context, commentId, 'success', {
    command,
    commentUser,
    approver,
    customArgs,
    runUrl,
    s3Url,
    bucketName
  });
}

/**
 * Handle preview generation failure comment update
 */
async function handlePreviewFailure(github, context, env) {
  const commentId = env.STATUS_COMMENT_ID;
  const commentUser = env.COMMENT_USER;
  const customArgs = env.CUSTOM_ARGS || '';
  const command = customArgs ? `/preview ${customArgs}` : '/preview';
  const approver = env.GITHUB_ACTOR;
  const runUrl = `https://github.com/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}`;

  await safeUpdateComment(github, context, commentId, 'failure', {
    command,
    commentUser,
    approver,
    customArgs,
    runUrl
  });
}

/**
 * Handle auto-cleanup comment creation
 */
async function handleAutoCleanup(github, context, env) {
  const prNumber = env.PR_NUMBER;
  const wasMerged = env.PR_MERGED === 'true';
  const runUrl = `https://github.com/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}`;
  const bucketName = env.BUCKET_NAME;

  await createAutoCleanupComment(github, context, prNumber, wasMerged, runUrl, { bucketName });
}

/**
 * Handle manual cleanup starting comment update
 */
async function handleCleanupStarting(github, context, env) {
  const commentId = env.STATUS_COMMENT_ID;
  const commentUser = env.COMMENT_USER;
  const approver = env.GITHUB_ACTOR;
  const runUrl = `https://github.com/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}`;

  await updateStatusComment(github, context, commentId, 'running', {
    command: '/preview cleanup',
    commentUser,
    approver,
    runUrl,
    isCleanup: true
  });
}

/**
 * Handle manual cleanup result comment update
 */
async function handleCleanupResult(github, context, env, jobStatus) {
  const commentId = env.STATUS_COMMENT_ID;
  const commentUser = env.COMMENT_USER;
  const approver = env.GITHUB_ACTOR;
  const runUrl = `https://github.com/${env.GITHUB_REPOSITORY}/actions/runs/${env.GITHUB_RUN_ID}`;
  const bucketName = env.BUCKET_NAME;
  const status = jobStatus === 'success' ? 'success' : 'failure';

  await safeUpdateComment(github, context, commentId, status, {
    command: '/preview cleanup',
    commentUser,
    approver,
    runUrl,
    bucketName,
    isCleanup: true
  });
}

/**
 * Handle cancelled/rejected preview comment update
 */
async function handleCancelledPreview(github, context, env, securityCheck) {
  const commentId = env.STATUS_COMMENT_ID;
  const commentUser = env.COMMENT_USER;
  const isCleanup = env.IS_CLEANUP === 'true';
  const customArgs = securityCheck.outputs.custom_args || '';
  const command = isCleanup ? '/preview cleanup' : customArgs ? `/preview ${customArgs}` : '/preview';

  if (!commentId) {
    console.log('No comment ID available');
    return;
  }

  try {
    await updateStatusComment(github, context, commentId, 'cancelled', {
      command,
      commentUser,
      customArgs,
      isCleanup
    });
    console.log(`Updated comment ${commentId} for cancelled ${isCleanup ? 'cleanup' : 'preview generation'}`);
  } catch (error) {
    console.error(`Failed to update comment: ${error.message}`);
  }
}

module.exports = {
  handleInitialComment,
  handlePendingApproval,
  handlePreviewStarting,
  handlePreviewSuccess,
  handlePreviewFailure,
  handleAutoCleanup,
  handleCleanupStarting,
  handleCleanupResult,
  handleCancelledPreview
};