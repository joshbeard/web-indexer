// GitHub API wrapper for comment operations
// Handles creating, updating, and managing PR/issue comments

const { getDeploymentStatusComment, getAutoCleanupComment } = require('./comment-templates');

/**
 * Create a new status comment on a PR
 */
async function createStatusComment(github, context, prNumber, status, options = {}) {
  const body = getDeploymentStatusComment(status, options);

  try {
    const { data: comment } = await github.rest.issues.createComment({
      issue_number: prNumber,
      owner: context.repo.owner,
      repo: context.repo.repo,
      body: body
    });

    console.log(`Created status comment: ${comment.id}`);
    return comment.id;
  } catch (error) {
    console.error(`Failed to create comment: ${error.message}`);
    throw error;
  }
}

/**
 * Update an existing status comment
 */
async function updateStatusComment(github, context, commentId, status, options = {}) {
  if (!commentId) {
    console.log('No comment ID available for update');
    return;
  }

  const body = getDeploymentStatusComment(status, options);

  try {
    await github.rest.issues.updateComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      comment_id: commentId,
      body: body
    });

    console.log(`Updated comment ${commentId} - status: ${status}`);
  } catch (error) {
    console.error(`Failed to update comment: ${error.message}`);
    throw error;
  }
}

/**
 * Create auto-cleanup comment on PR close/merge
 */
async function createAutoCleanupComment(github, context, prNumber, wasMerged, runUrl, options = {}) {
  const body = getAutoCleanupComment({ prNumber, wasMerged, runUrl, ...options });

  try {
    const { data: comment } = await github.rest.issues.createComment({
      issue_number: prNumber,
      owner: context.repo.owner,
      repo: context.repo.repo,
      body: body
    });

    console.log(`Posted cleanup comment on PR #${prNumber}`);
    return comment.id;
  } catch (error) {
    console.error(`Failed to create cleanup comment: ${error.message}`);
    throw error;
  }
}

/**
 * Safe comment update with error handling
 */
async function safeUpdateComment(github, context, commentId, status, options = {}) {
  try {
    await updateStatusComment(github, context, commentId, status, options);
  } catch (error) {
    console.error(`Safe comment update failed: ${error.message}`);
    // Don't fail the workflow on comment errors
  }
}

module.exports = {
  createStatusComment,
  updateStatusComment,
  createAutoCleanupComment,
  safeUpdateComment
};
