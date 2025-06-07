// GitHub comment templates and API helpers
// Used by preview workflow for consistent comment formatting

/**
 * Generate web-indexer preview status comment body
 */
function getDeploymentStatusComment(status, options = {}) {
  const {
    command = '/preview',
    commentUser,
    approver,
    customArgs,
    runUrl,
    isCleanup = false,
    s3Url,
    bucketName
  } = options;

  const emoji = isCleanup ? '🧹' : '🗂️';
  const action = isCleanup ? 'Cleanup' : 'Web-Indexer Preview';

  let body = `## ${emoji} ${action} Status\n\n` +
             `**Command:** \`${command}\`\n` +
             `**Requested by:** @${commentUser}\n`;

  if (approver) {
    body += `**Approved by:** @${approver}\n`;
  }

  body += `**Status:** ${getStatusEmoji(status)} ${getStatusText(status, isCleanup)}\n\n`;

  switch (status) {
    case 'queuing':
      body += `🔄 Queuing ${isCleanup ? 'cleanup' : 'preview generation'} request...\n\n`;
      break;

    case 'pending':
      body += `🔒 Waiting for authorized reviewer to approve this ${isCleanup ? 'cleanup' : 'preview generation'}...\n\n`;
      break;

    case 'running':
      body += `⚡ ${action} is now running with full access to AWS resources\n\n`;
      break;

    case 'success':
      body += `🎉 **${action} Complete!**\n`;

      if (!isCleanup) {
        body += `- **Live Preview:** [${s3Url}](${s3Url})\n` +
                `- **S3 Bucket:** \`${bucketName}\`\n` +
                `- **Arguments:** \`${customArgs || '(all themes)'}\`\n` +
                `- **Artifacts:** [Download results](${runUrl})\n` +
                `- **Logs:** [View workflow details](${runUrl})\n\n` +
                `**Preview includes:**\n` +
                `✅ Responsive web-indexer interface\n` +
                `✅ All configured themes (Default, Solarized, Nord, Dracula)\n` +
                `✅ Recursive directory indexing\n` +
                `✅ Dark mode support\n\n`;
      } else {
        body += `- **S3 Bucket:** \`${bucketName}\` (removed)\n` +
                `- **Logs:** [View cleanup details](${runUrl})\n\n` +
                `**Cleanup completed:**\n` +
                `✅ S3 bucket and all objects removed\n` +
                `✅ Preview environment resources cleaned up\n` +
                `✅ Temporary artifacts removed\n\n`;
      }
      break;

    case 'failure':
      body += `🚨 **${action} Error**\n` +
              `The ${isCleanup ? 'cleanup' : 'preview generation'} encountered an error during execution.\n\n` +
              `- **Arguments:** \`${customArgs || '(default)'}\`\n` +
              `- **Error Logs:** [View workflow details](${runUrl})\n` +
              `- **Debug Info:** Check the workflow logs for detailed error information\n\n`;
      break;

    case 'cancelled':
      body += `🚫 **${action} Not Approved**\n` +
              `The ${isCleanup ? 'cleanup' : 'preview generation'} was either:\n` +
              `- Rejected by an authorized reviewer\n` +
              `- Cancelled before approval\n` +
              `- Timed out waiting for approval\n\n` +
              `To retry, post the command again in a new comment.\n\n`;
      break;
  }

  body += `---\n` + getFooterText(status, isCleanup);

  return body;
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