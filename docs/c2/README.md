# Terraform C2 Provider

Адаптация [Terraform AWS Provider](https://github.com/hashicorp/terraform-provider-aws) от **HashiCorp**
под **Облако КРОК** (C2).

- [Требования](#требования)
- [Общая информация](#общая-информация)
- [Начало работы](#начало-работы)
    - [Установка линтеров](#установка-линтеров)
- [aws-sdk-go](#aws-sdk-go)
    - [Изменение версии aws-sdk-go](#изменение-версии-aws-sdk-go)
- [Тесты](#тесты)
    - [Unit](#unit)
    - [Acceptance](#acceptance)
- [Документация](#документация)
    - [Запуск линтеров](#запуск-линтеров)
- [Выпуск релиза](#выпуск-релиза)
    - [Настройка окружения](#настройка-окружения)
    - [Версионирование](#версионирование)
    - [Релиз](#релиз)
- [Публикация провайдера в официальном terraform registry](#публикация-провайдера-в-официальном-terraform-registry)
- [Публикация провайдера в private terraform registry](#публикация-провайдера-в-private-terraform-registry)
    - [Структура s3 бакета](#структура-s3-бакета)
    - [Загрузка новой версии](#загрузка-новой-версии)
        - [Использование скрипта](#использование-скрипта)
- [Использование провайдера](#использование-провайдера)
- [TODO](#todo)

## Требования

- [Terraform](https://www.terraform.io/downloads.html) 0.13+ (запуск приемочных тестов)
- [Go](https://golang.org/doc/install) 1.17+ (сборка провайдера)
- [Docker](https://docs.docker.com/get-docker/) (запуск линтеров для документации)

## Общая информация

**Провайдер** - это плагин для [Terraform](https://www.terraform.io/), который реализует возможность управления
ресурсами некоего сервиса (например, облака или БД) через его API. Для каждого ресурса в провайдер добавляется
схема и CRUD операции. Информация об API предоставляется в виде отдельного модуля.

**Terraform C2 Provider** реализуется на базе **Terraform AWS Provider**.
Также создан форк модуля с AWS API: [C2Devel/aws-sdk-go](https://github.com/C2Devel/aws-sdk-go).

Для публикации провайдера в официальном [terraform registry](https://registry.terraform.io/) под новым именем
(ранее - *aws*) форк переименован в  
*terraform-provider-croccloud*.

Опубликованный провайдер: https://registry.terraform.io/providers/C2Devel/croccloud

## Начало работы

Для работы с провайдером требуется установка [go](https://golang.org/doc/install) (см. [требования](#требования)).

Клонирование репозитория и сборка провайдера:

```
$ git clone git@github.com:C2Devel/terraform-provider-croccloud.git && cd terraform-provider-croccloud
...
$ make build
```

После сборки артефакт `terraform-provider-aws` будет доступен в директории `$GOPATH/bin`.

Установка переменной окружения `$GOPATH`:

```
$ export GOPATH=$(go env GOPATH)
$ ls $GOPATH/bin/terraform-provider-aws
...
```

**Важно!** `make build` использует команду `go install`, которая не позволяет изменить
имя артефакта - `terraform-provider-aws`. Используется только для dev сборок.

Для сборки проекта также можно использовать команду:

```
$ go build -o terraform-provider-croccloud
```

Артефакт `terraform-provider-croccloud` будет создан в директории запуска.

### Установка линтеров

**Опционально.** Установка дополнительных библиотек (линтеры, форматтеры и т.д.): `make tools`

Артефакты также будут находиться в `$GOPATH/bin`.

**Важно!** Для запуска связанных `make` таргетов требуется добавить путь `$GOPATH/bin` в `$PATH`.

Таргеты для запуска линтеров для кода:

```
$ make lint
$ make semgrep
```

## aws-sdk-go

Обращения к модулю **aws-sdk-go** (AWS API) перенаправлены на [форк](https://github.com/C2Devel/aws-sdk-go)
с помощью директивы `replace` в `go.mod`.

Для локальной разработки можно указать в правой части директивы путь к директории с исходным кодом модуля.

```
# go.mod
...
replace github.com/aws/aws-sdk-go => <path-to-aws-sdk-go>
```

**Важно!** Можно тэггировать изменения модуля и обновлять в `go.mod` тэг, но его нельзя будет повторно использовать,
т.к. go не позволяет изменять версии после их публикации.

**Важно!** Форк **aws-sdk-go** нельзя вытянуть через `go get github.com/C2Devel/aws-sdk-go@1.44.10`.

### Изменение версии aws-sdk-go

1. Обновление тега `github.com/C2Devel/aws-sdk-go`:

   ```
   # go.mod
   ...
   replace github.com/aws/aws-sdk-go => github.com/C2Devel/aws-sdk-go v1.44.10-new
   ```

2. **Опционально.** Если изменилась upstream версия: обновление тега `github.com/aws/aws-sdk-go` в блоке `require`.  
   Не влияет на сборку.
3. Обновление зависимостей: `go mod tidy`

**Важно!** `go mod tidy` актуализирует все зависимости в `go.mod`, т.е. итоговые изменения могут касаться
не только **aws-sdk-go**.

## Тесты

В проекте есть два типа тестов: **unit** и **acceptance**. Они лежат рядом с функционалом (файлы: *_test.go).

Тесты написаны с помощью [go testing](https://go.dev/doc/code#Testing), для приемочных дополнительно используется
пакет [acctest](../../internal/acctest/acctest.go).

### Unit

Запуск unit тестов: `make test`

### Acceptance

Для запуска приемочных тестов требуется установка [Terraform](https://golang.org/doc/install)
(см. [требования](#требования)).

**Важно!** Тесты используют реальные облачные ресурсы. **Требуется доработка тестов для запуска на C2.**

По запуску и написанию приемочных тестов есть
[документация](../../docs/contributing/running-and-writing-acceptance-tests.md).

Команды:

- запуск всех тестов: `make testacc`
- запуск конкретного теста: `make testacc TESTS=TestAccEC2EBSVolume_basic PKG=ec2`

## Документация

Информация о провайдере расположена в директориях:

- `docs/` - инструкции для разработки, roadmap;
- `website/` - документация к провайдеру, которая публикуется в официальном terraform registry
  ([инструкция](https://www.terraform.io/registry/providers/docs) по документированию от **Terraform**).

Структура директории website:

```
website/
|-- docs/ 
|    |-- d/                          # набор описаний для terraform data sources
|    |-- guides/
|    |-- r/                          # набор описаний для terraform resources
|    |    |-- <resource>.html.markdown
|    |    |-- ...
|    |
|    |-- index.html.markdown         # стартовая страница
|-- allowed-subcategories.txt        # разделы документации
```

### Запуск линтеров

Для запуска линтеров требуется установка [Docker](https://golang.org/doc/install) и собственно линтеров
(см. [установка линтеров](#установка-линтеров)).

`docs/`: проверяется форматирование markdown файлов и ошибки в тексте (English).

```
$ make docs-lint
```

`website/`: проверяется форматирование markdown файлов, ошибки в тексте (English)
и соответствие документации спецификации terraform registry.

```
$ make website-lint
...
$ make docscheck
```

## Выпуск релиза

Релиз провайдера формируется в соответствии с требованиями к публикации в terraform registry.

Подготовка релизных артефактов (сборка под разные архитектуры и ОС, подпись, архивация и т.д.) описана
в `.goreleaser.yml`

Дополнительная информация:
[инструкция](https://www.terraform.io/registry/providers/publishing#creating-a-github-release)
по созданию релиза провайдера на github от **Terraform**.

### Настройка окружения

1. Установка [goreleaser](https://goreleaser.com/install/)

   ```
   $ go install github.com/goreleaser/goreleaser@latest
   ...
   $ export GOBIN=$(go env GOPATH)/bin
   $ $GOBIN/goreleaser -v
   ...
   ```

2. Создание **GPG** ключа
   ([инструкция от github](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key))
    - **Важно!** Ключ должен быть без пароля
    - **Опционально.** Привязка ключа к github аккаунту
3. Установка переменной `GPG_FINGERPRINT`

   ```
   $ gpg --list-secret-keys --keyid-format LONG
   ...
   sec   rsa4096/<id> 2022-04-19 [SC]
   ...
   $ gpg --list-secret-keys --with-colons --fingerprint <id> | grep fpr | cut -f 10 -d :
   <fingerprint>

   $ export GPG_FINGERPRINT=<fingerprint>
   ```

4. Создание **Personal Access Token** с разрешением **public_repo**
   ([инструкция от github](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token))
    - **Важно!** Скопировать сгенерированный токен можно только сразу после создания
5. Установка переменной `GITHUB_TOKEN`

   ```
   $ export GITHUB_TOKEN=<generated token by github>
   ```

### Версионирование

Версии провайдера должны соответствовать спецификации [Semantic Version](https://semver.org/) и начинаться с **v**
(например, **v1.2.3** или **v1.2.3-pre**). Версия провайдера фиксируется только в виде тэга.

**Важно!** Не допускается обновление уже выпущенных версий, т.к. могут возникнуть проблемы со скачиванием провайдера
из terraform registry.

Схема версионирования C2: **<upstream_version>-CROCX**, X - инкрементируется каждый релиз.

### Релиз

**Важно!** Релизы провайдера выпускаются с ветки **develop** (установлена дефолтной).
Ветка **main** используется для получения обновлений с upstream.

1. **Опционально.** Если требуется обновить версию **aws-sdk-go**,
   см. [изменение версии aws-sdk-go](#изменение-версии-aws-sdk-go)
2. Запуск тестов (см. [тесты](#тесты))
3. **Опционально.** Включение автопубликации релиза в github (флаг `release.draft` в `.goreleaser.yml`)

   ```
   # .goreleaser.yml
   ...
   release:
     draft: false
   ```

   По умолчанию релиз будет опубликован в origin.

4. Подготовка репозитория: на релизной ветке не должно быть незакоммиченных изменений, untracked файлов
5. Установка релизного тега с версией (см. [версионирование](#версионирование)) и его публикация

   ```
   $ git tag v1.2.3-CROC0
   $ git push <remote> v1.2.3-CROC0
   ```

6. Сборка и подпись релизных артефактов. Артефакты будут размещены в директории `dist/`

   ```
   $ $GOBIN/goreleaser release --rm-dist
   ```

7. **Опционально.** Если на шаге 2 **не** была включена автопубликация (`release.draft: true`):
   создание релиза на github и загрузка артефактов:
    - `dist/terraform-provider-croccloud_{VERSION}_{OS}_{ARCH}.zip`
        - Для всех архитектур и ОС.
    - `dist/terraform-provider-croccloud_{VERSION}_SHA256SUMS`
    - `dist/terraform-provider-croccloud_{VERSION}_SHA256SUMS.sig`
    - `terraform-provider-croccloud_{VERSION}_manifest.json`
        - Файл создается вручную:
        - `cp terraform-registry-manifest.json terraform-provider-croccloud_{VERSION}_manifest.json`

## Публикация провайдера в официальном terraform registry

Дополнительная информация:
[инструкция](https://www.terraform.io/registry/providers/publishing#publishing-to-the-registry)
по публикации от **Terraform**.

**Важно!** Для публикации провайдера в terraform registry требуется хотя бы одна выпущенная версия.

Порядок публикации:

1. Регистрация в [terraform registry](https://registry.terraform.io/) с помощью github аккаунта
2. Добавление **GPG** ключа, который использовался для подписи релиза
    (см. [настройка окружения](#настройка-окружения), шаг 2) в [настройках профиля](https://registry.terraform.io/settings/gpg-keys)

   ```
   $ gpg --list-secret-keys --keyid-format LONG
   ...
   sec   rsa4096/<id> 2022-04-19 [SC]
   ...
   $ gpg --armor --export <id>
   -----BEGIN PGP PUBLIC KEY BLOCK-----
   ...
   ```

3. Выбор провайдера в меню [Publish -> Provider](https://registry.terraform.io/publish/provider) и публикация

После публикации провайдера для репозитория будет создан webhook на события из группы `Releases`.
Новые релизы будут автоматически подтянуты в registry.

**Важно!** Terraform не позволяет самостоятельно удалять опубликованный провайдер или одну из его версий.
Не допускается обновление уже выпущенных версий.

Опубликованный провайдер: https://registry.terraform.io/providers/C2Devel/croccloud

## Публикация провайдера в private terraform registry

Terraform registry может быть организован в виде s3 бакета.

**Важно!** У бакета должен быть настроен доступ по https
([инструкция от CROC](https://docs.cloud.croc.ru/en/services/object_storage/instructions.html#filestorage-https-for-website-buckets)).
При включении web-доступа в качестве индексной страницы требуется указать `index.json`.

Согласно [описанию](https://www.terraform.io/internals/provider-registry-protocol) протокола,
terraform registry содержит в себе файлы с версиями провайдеров и метаинформацией для каждой сборки,
в которой указаны ссылки на конкретные артефакты.

### Структура s3 бакета

```
.well-known/
|-- terraform.json                                 # служебный файл

providers/
|-- c2devel/                                       # разработчик
     |-- croccloud/                                # имя провайдера
          |-- 1.0.0-CROC0/
          |    |-- download/
          |         |-- linux/
          |         |    |-- amd64/
          |         |    |    |-- index.json       # метаинформация для сборки 1.0.0-CROC0_linux_amd64
          |         |    |-- ...
          |         |-- ...
          |    
          |-- versions/         
               |-- index.json                      # версии провайдера
```

`.well-known/terraform.json` используется при первом обращении к registry для проверки доступности.
Также в нем указывается базовый url для всех провайдеров.

```
# .well-known/terraform.json

{
  "providers.v1": "/providers/"
}
```

Исходный вид файла с версиями провайдера:

```
# providers/c2devel/croccloud/versions/index.json

{
    "id": "c2devel/croccloud",
    "versions": [],
    "warnings": null
}
```

### Загрузка новой версии

После релиза новой версии провайдера (см. [релиз](#релиз)) артефакты будут доступны в директории `dist/`.

Для загрузки версии в s3 бакет требуется сформировать файлы с версиями и метаинформацией.
Их можно получить из официального terraform registry, если провайдер уже опубликован,
или сделать самостоятельно в соответствии с [протоколом](https://www.terraform.io/internals/provider-registry-protocol).

Если требуется создание s3 бакета и директорий, см. [структура s3 бакета](#структура-s3-бакета).

Порядок загрузки:

1. Получение версий провайдера. В файле должна присутствовать загружаемая версия

   ```
   $ curl https://registry.terraform.io/v1/providers/c2devel/croccloud/versions --output versions.json
   ```

   В блоке `versions.<version>.platforms` указаны архитектуры и ОС, под которые версия собиралась.
2. Получение метаинформации для выбранных сборок провайдера

   ```
   $ curl https://registry.terraform.io/v1/providers/c2devel/croccloud/<version>/download/<os>/<arch> --output <version>_<os>_<arch>.json
   ```

3. **Опционально.** Сохранение артефактов в собственное хранилище.
   Артефакты можно скачать по ссылкам в метаинформации или скопировать из директории `dist/`:
   - `dist/terraform-provider-croccloud_{VERSION}_{OS}_{ARCH}.zip`
      - Для всех архитектур и ОС.
   - `dist/terraform-provider-croccloud_{VERSION}_SHA256SUMS`
   - `dist/terraform-provider-croccloud_{VERSION}_SHA256SUMS.sig`

4. **Опционально.** Если был выполнен шаг 3: обновление ссылок в метаинформации
5. Обновление s3 бакета:
   - обновление файла с версиями: `version.json` -> `providers/c2devel/croccloud/versions/index.json`
   - загрузка метаинформации для сборок версии:  
     `<version>_<os>_<arch>.json` -> `providers/c2devel/croccloud/<version>/download/<os>/<arch>/index.json`
   **Важно!** Файлы должны быть загружены с mime-типом "application/json". Для файлов должен быть
   открыт доступ на чтение без аутентификации.

#### Использование скрипта

Для автоматизации процесса загрузки новых версий в s3 бакет из официального terraform registry
можно использовать [bash скрипт](../../scripts/update-s3-registry.sh).

Скрипт анализирует файлы с версиями провайдера в s3 бакете и в официальном registry
и для всех версий, которые отсутствуют в бакете, загружает из официального registry файлы с
метаинформацией для сборок.

**Важно!** Файл с версиями в s3 бакете будет приведен к виду официального registry.

Для запуска скрипта требуется установка и настройка утилиты [s3cmd](https://s3tools.org/s3cmd)
([инструкция от CROC](https://docs.cloud.croc.ru/ru/api/tools/s3cmd.html?highlight=s3cmd))
и созданный s3 бакет (см. [структура s3 бакета](#структура-s3-бакета)).

Переменные окружения скрипта:

- `TF_REGISTRY_URL` - url terraform registry, по умолчанию: `"https://registry.terraform.io/"`
- `S3_REGISTRY_URL` - url s3 registry, обязательно
- `S3_BUCKET_NAME` - имя бакета, обязательно
- `PROVIDER_NAME` - имя провайдера, по умолчанию: `"c2devel/croccloud"`
- `S3_BACKUP_DIR` - директория для бэкапа бакета, опционально. Если директория не указана,
   бэкап сделан не будет

Запуск скрипта:

```
$ cd scripts
$ ./update-s3-registry.sh
...
```

## Использование провайдера

Провайдер в terraform registry: https://registry.terraform.io/providers/C2Devel/croccloud

Примеры использования **Terraform** для C2: [C2Devel/terraform-examples](https://github.com/C2Devel/terraform-examples)

Конфигурация провайдера **C2Devel/croccloud** после его публикации в официальном terraform registry:

```
# provider.tf

terraform {
  required_providers {
    aws = {
      # case-insensistive
      source = "c2devel/croccloud"
      version = "4.14.0-CROC1"
    }
  }
}
 
provider "aws" {
  # Configuration options
}
```

**Важно!** В конфигурации в качестве имени провадйера используется `aws`, т.к. сохранена схема
именования terraform ресурсов: **aws_***. Если используется другое имя (например, `croccloud`),
**Terraform** автоматически попытается загрузить провайдер **hashicorp/aws**.

Если провайдер опубликован в s3 бакете, в поле `source` добавляется url бакета без схемы. Например,  
`tf-registry.croc.ru/c2devel/croccloud`.

Также можно локально собрать артефакт (см. [начало работы](#начало-работы)) и настроить **Terraform (v0.14+)**
на его использование. По умолчанию конфигурация **Terraform** находится в файле `~/.terraformrc`.
Для разработки можно создать отдельную конфигурацию.

Пример dev конфигурации `dev.tfrc`, в которой обращение к провайдеру `c2devel/<name>` перенаправлено в директорию,
в которой находится артефакт `terraform-provider-<name>`:

```
# dev.tfrc

provider_installation {
  dev_overrides {
    "c2devel/<name>" = "<absolute-path-to-artifact-dir>"
  }
  direct {}
}
```

Установка `dev.tfrc` в качестве **Terraform** конфигурации:

```
$ export TF_CLI_CONFIG_FILE=<path-to-dev.tfrc>
```

**Важно!** Сначала для проекта должен быть выполнен `terraform init` (установлены провайдеры, сформирован lock файл),
а потом уже добавлено перенаправление на свой артефакт. Повторно запускать `terraform init` не требуется и
не рекомендуется, т.к. **Terraform** будет пытаться установить все провайдеры из официального registry, в том числе и
переопределенный. Это может привести к ошибкам.

## TODO

1. Настройка github actions: прогон тестов, прогон линтеров, сообщения в PR
2. Доработка acceptance тестов для запуска на C2
3. Использовать в `make build` команду `go build` вместо `go install` для того,
   чтобы иметь возможность задать имя артефакта
4. Обновить схему именования ресурсов: **aws_*** -> **croccloud_***,
   чтобы иметь возможность использовать в конфигурации в качестве имени провайдера `croccloud`.
   Потребуется проверка совместимости с aws конфигурациями
