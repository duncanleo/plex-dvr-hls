const Mustache = require('mustache');
const fs = require('fs');
const dateFormat = require('dateformat');

const channels = require('../../channels.json');

const xmltv = (req, res) => {
  const today = new Date();
  res.send(
    Mustache.render(
      fs.readFileSync('templates/xmltv.mustache', { encoding: 'utf8' }),
      {
        channels: channels.map((c, index) => ({
          id: index + 1,
          name: c.name,
        })),
        programmes: [...Array(24).keys()].map((hour) => {
          today.setHours(hour);
          today.setMinutes(0);
          today.setSeconds(0);
          const dateTimeStart = dateFormat(today, 'yyyymmddHHmmss o');
          const hourStr = dateFormat(today, 'htt');
          today.setHours(hour + 1);
          today.setMinutes(59);
          today.setSeconds(59);
          const dateTimeEnd = dateFormat(today, 'yyyymmddHHmmss o');
          return { hourStr, dateTimeStart, dateTimeEnd };
        }),
      },
    ),
  );
};

module.exports = {
  xmltv,
};
