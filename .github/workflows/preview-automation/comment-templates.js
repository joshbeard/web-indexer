// GitHub comment templates and API helpers
// Used by preview workflow for consistent comment formatting

/**
 * Generate minimal web-indexer preview status comment
 */
function getDeploymentStatusComment(status, options = {}) {
  const {
    isCleanup = false,
    s3Url
  } = options;

  const cleanupEmoji = '🧹';
  const previewEmoji = '🗂️';

  switch (status) {
    case 'queuing':
      return `${isCleanup ? cleanupEmoji : previewEmoji} ${isCleanup ? 'Cleanup' : 'Preview'} generating...`;

    case 'pending':
      return `⏳ ${isCleanup ? 'Cleanup' : 'Preview'} awaiting approval...`;

    case 'running':
      return `⚡ ${isCleanup ? 'Cleanup' : 'Preview'} approved, generating...`;

    case 'success':
      if (isCleanup) {
        return `✅ Preview environment cleaned up`;
      } else {
        return `✅ Preview ready: [View Demo](${s3Url})`;
      }

    case 'failure':
      return `❌ ${isCleanup ? 'Cleanup' : 'Preview'} failed`;

    case 'cancelled':
      return `🚫 ${isCleanup ? 'Cleanup' : 'Preview'} not approved`;

    default:
      return `❓ ${isCleanup ? 'Cleanup' : 'Preview'} status unknown`;
  }
}

/**
 * Generate auto-cleanup comment body
 */
function getAutoCleanupComment(options = {}) {
  const { prNumber, wasMerged, runUrl, bucketName } = options;

  return `## 🧹 Web-Indexer Preview Cleanup\n\n` +
         `**Trigger:** PR ${wasMerged ? 'merged' : 'closed'}\n` +
         `**Status:** ✅ Preview environment automatically cleaned up\n\n` +
         `🎉 **Cleanup Complete!**\n` +
         `- **S3 Bucket:** \`${bucketName}\` (removed)\n` +
         `- **Preview URL:** No longer accessible\n` +
         `- **Artifacts:** Temporary deployment artifacts removed\n` +
         `- **Logs:** [View cleanup details](${runUrl})\n\n` +
         `**Auto-cleanup verified:**\n` +
         `✅ S3 bucket and all objects removed\n` +
         `✅ Preview environment resources cleaned up\n` +
         `✅ Temporary artifacts removed\n` +
         `✅ Environment reset for future deployments\n\n` +
         `---\n` +
         `*Cleanup completed automatically on PR ${wasMerged ? 'merge' : 'close'}*`;
}

function getStatusEmoji(status) {
  const emojis = {
    queuing: '🔄',
    pending: '⏳',
    running: '🔄',
    success: '✅',
    failure: '❌',
    cancelled: '❌'
  };
  return emojis[status] || '❓';
}

function getStatusText(status, isCleanup = false) {
  const action = isCleanup ? 'cleanup' : 'preview';
  const texts = {
    queuing: `Queuing ${action} request...`,
    pending: `Pending approval to \`preview-pr\` environment`,
    running: `Approved! ${isCleanup ? 'Cleaning up' : 'Generating preview in'} \`preview-pr\` environment...`,
    success: `Successfully ${isCleanup ? 'cleaned up' : 'generated preview in'} \`preview-pr\` environment`,
    failure: `${isCleanup ? 'Cleanup' : 'Preview generation'} failed`,
    cancelled: `${isCleanup ? 'Cleanup' : 'Preview generation'} cancelled or rejected`
  };
  return texts[status] || 'Unknown status';
}

function getFooterText(status, isCleanup = false) {
  const action = isCleanup ? 'cleanup' : 'preview generation';
  const footers = {
    queuing: '*This comment will be updated with progress*',
    pending: '*This comment will be updated with progress*',
    running: '*This comment will be updated with progress*',
    success: `*${isCleanup ? 'Cleanup' : 'Preview generation'} completed successfully*`,
    failure: `*${isCleanup ? 'Cleanup' : 'Preview generation'} failed - check logs for details*`,
    cancelled: `*${isCleanup ? 'Cleanup' : 'Preview generation'} was not executed*`
  };
  return footers[status] || '*Status unknown*';
}

module.exports = {
  getDeploymentStatusComment,
  getAutoCleanupComment
};