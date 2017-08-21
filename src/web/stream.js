const ffmpeg = require('fluent-ffmpeg');

const channels = require('../../channels.json');

const stream = (req, res) => {
  const { channelNum } = req.params;
  const { transcode } = req.query;
  const channel = channels[channelNum - 1];

  let ffmpegStream = ffmpeg(channel.url)
    .videoCodec('libx264')
    .audioCodec('aac')
    .addOutputOption('-copyinkf')
    .addOutputOption('-metadata service_provider=AMAZING')
    .addOutputOption(`-metadata service_name=${channel.name.replace(/\s/g, '-')}`)
    .addOutputOption('-tune zerolatency')
    .addOutputOption('-mbd rd')
    .addOutputOption('-flags +ilme+ildct')
    .addOutputOption('-fflags +genpts')
    .outputFormat('mpegts');

  switch (transcode) {
    case 'mobile':
    case 'internet720':
      ffmpegStream = ffmpegStream
        .size('1280x720')
        .outputFPS(30);
      break;
    // case 'internet480':
    //   break;
    // case 'internet360':
    //   break;
    // case 'internet240':
    //   break;
    default:
      break;
  }

  ffmpegStream.on('error', console.error);

  ffmpegStream.pipe(res);
};

module.exports = {
  stream,
};
