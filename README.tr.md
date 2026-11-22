# ❄️ SnowDesk

Windows masaüstünüze yağan kar, kardan adamlar ve pencerelerin üstünde biriken kar.
Küçük, bağımlılıksız, açık kaynak bir kış efekti.

## Özellikler

- Rüzgarlı, derinlik hissi veren kar yağışı
- **Kar her pencerenin ve görev çubuğunun üstünde birikir**, pencereyi sürükleyince kar savrulup dökülür, önde başka pencere varsa arkada kalan karı gizler
- İsteğe bağlı kardan adamlar (varsayılan kapalı, tray menüsünden açılır)
- **Tıklamayı geçirir**: fareyi ve odağı asla engellemez
- Tam ekran oyun/video açılınca kendini gizler
- Tray menüsü: duraklat, kardan adam aç/kapat, yoğunluk, karı temizle, *Windows ile başlat*
- Tek bir ~2 MB `.exe`; kurulum, yönetici izni, internet gerekmez

## Kurulum

[Releases](../../releases) sayfasından `snowdesk-windows-amd64.exe` dosyasını indirip çalıştırın.
Sağ alttaki kar tanesi simgesine sağ tıklayarak ayarlara ulaşın. Windows 10 / 11.

## Kaynaktan derleme

Go 1.21+ yeterli (herhangi bir işletim sisteminden derlenir):

```sh
make         # dist/ klasörüne .exe üretir
make test    # testler
```

Ayarlar `%AppData%\SnowDesk\config.json` dosyasında tutulur. Komut satırı seçenekleri için [README.md](README.md) dosyasına bakın.

## Lisans

MIT
