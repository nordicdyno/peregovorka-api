== регистрация текущего юзера  ==

* POST /users/register
in: нет
out:
в случае успеха – отдает 200
если юзер не залогинен - 404
в остальных случаях - 500 ("неизвестная ошибка")

== список всех доступных пользователей ==

* GET /users/list

in: нет
out:

в случае проблем не 200-й ответ
если все ок, то json вида
[
  {
    "uid":"71440567",
    "last_login_dt":"2014-11-21T22:23:43.877+03:00",
    "online":true,
    "name":"Александр Орловский",
    "avatar_url":"http://s5o.ru/storage/simple/ru/ugc/91/32/79/91/ruu27a38d6c82.113.48x48.jpeg"
  },
  {
    "uid":"158821431",
    "last_login_dt":"2014-11-21T22:24:05.595+03:00",
    "online":true,
    "name":"vovmos",
    "avatar_url":"http://s5o.ru/common/images/blank_icons/user_small.png"
  }
]

== добавить переговорку ==

* POST /rooms/create

in: guid - id пользователя для которого
out:
в случае проблем 500-ка и мусор в ответе

в случае успеха, 200-й код и json вида:
{
    "roomid": ""
}
