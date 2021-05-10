#### Общее описание

Сервис OTRS-time-accounting предназначен для быстрого доступа к данным OTRS, недоступным или труднодоступным при использовании стандартного WEB интерфейса системы.
Сейчас доступна статистика по списанному времени, а также краткая статистика по текущему количеству заявок.

#### Установка

В одну директорию поместить исполняемый файл, конфигурационный файл, папку с HTTP шаблонами ("website") и, при необходимости, БД файл со старыми данными. 

Пример содержимого директории:
```
.\OTRS-time-accounting
    OTRS_time_accaunting_build.exe
    config.yaml
    sqlite.db
    website
        layouts
            footer.html
            master.html
        favicon.ico
        index.html
        week.html
```

#### Использование

- Сервис запускается с помощью единственного исполняемого файла без указания каких-либо аргументов.
- Необходимым условием работы сервиса является доступность БД OTRS.
- В статистике за неделю учитывается
  - Списанное время в течении рабочего дня и помеченное как переработки (если есть, указывается в скобках).
    При этом, осуществляется подсветка цветом в зависимости от соответствия норме.
  - Списание времени за неделю (сумма за все дни недели, включая переработки)
    При этом, осуществляется подсветка цветом в зависимости от соответствия норме, с учётом количества рабочих дней в отображаемой неделе.
- По умолчанию определение рабочих и нерабочих дней жёстко привязано к дням недели (понедельник - пятница рабочие, суббота и воскресенье - выходные).
  - Предусмотрен механизм переопределения типа дня (рабочий в выходной и наоборот). включить или выключить переопределение типа дня можно с помощью соответствующих POST и DELETE запросов.
    ```
    http://localhost:9090/workingDayOverride?day=2021.05.08
    ```
    Где "localhost" и "9090" заменяются на хост и порт, используемые сервисом, а время указывается в формате "ГГГГ.ММ.ДД".

#### Особенности сервиса

- В качестве постоянного хранилища используется встроенная БД на основе sqlite3.
  - При отсутствии БД файла он создаётся автоматически.
- Может быть запущен как на Windows так и Unix системах.
- Предполагается свободный доступ к статистике для всех, у кого есть ссылка.