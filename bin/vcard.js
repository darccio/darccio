var vCard = require('vcards-js');

vc = vCard();
vc.firstName = 'Dario';
vc.lastName = 'Castañé';
vc.birthday = new Date('23-09-1985');
vc.title = 'Cloud Architect & Full Stack Developer';
vc.url = 'https://dario.im';
vc.gender = 'M';
vc.cellPhone = '+34665269218';
vc.email = 'i@dario.im';

vc.workPhone = '+34938055051';
vc.workUrl = 'https://engisoftcloud.com';
vc.workEmail = 'r.castane@engisoft.com';
vc.workAddress.street = 'Av. Europa, 7 L-8';
vc.workAddress.city = 'Igualada';
vc.workAddress.postalCode = '08700';
vc.workAddress.stateProvince = 'Barcelona';
vc.workAddress.countryRegion = 'Catalonia';

vc.socialUrls['github'] = 'https://github.com/imdario';
vc.socialUrls['linkedin'] = 'https://www.linkedin.com/in/im.dario';
vc.socialUrls['telegram'] = 'https://t.me/im_dario';
vc.socialUrls['twitter'] = 'https://twitter.com/im_dario';

vc.logo.embedFromFile('./static/images/default.png');
vc.photo.embedFromFile('./static/images/logo.png');

vc.source = 'https://dario.im/.well-known/carddav';

vc.saveToFile('./static/dario-castane.vcf');