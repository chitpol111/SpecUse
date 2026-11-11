# หลักฐานผู้ผลิตและการใช้ใน proposal

ตรวจแหล่งข้อมูลวันที่ 2026-11-11 ใช้เฉพาะเอกสารผู้ผลิตโดยตรง
ข้อมูลด้านล่างรองรับว่าต้องเก็บ/ตรวจอะไร ไม่ใช่ specs ของสินค้าใน SpecUse
กฎของรุ่นที่อ้างอิงใช้ได้เฉพาะรุ่น/เงื่อนไขนั้น ไม่ถูกใช้แทน default ของ motherboard/GPU/Case ทุกรุ่น
ห้ามนำตัวอย่างชื่อรุ่นในหน้านี้ไปสร้าง catalog product หรือคัดลอกตัวเลขเข้า fixture อัตโนมัติ

| ID | เอกสาร/ตำแหน่งที่อ่าน | ข้อเท็จจริงที่รองรับ | ผลต่อการออกแบบของ SpecUse |
|---|---|---|---|
| S1 | [Kingston: SATA/M.2 SSD FAQ](https://www.kingston.com/en/ssd/ssd-faq), หัวข้อ M.2 keys, lengths, SATA-only vs PCIe, x4 drive on x2 port | M.2 ต้องพิจารณารูปทรง/key/ขนาดร่วมกับ protocol; PCIe/SATA ไม่ใช่แทนกันได้เพราะเสียบรูปทรงเดียวกัน; โหมด lanes ต่ำลงอาจทำงานที่ความเร็วลดลงได้ | แยก connector/form factor/connection_type; ตรวจข้อมูลที่เกี่ยวข้องครบ และไม่ใช้ความเร็วลดลงเป็น failure อัตโนมัติ |
| S2 | [ASUS PRIME B550-PLUS Tech Specs](https://www.asus.com/motherboards-components/motherboards/prime/prime-b550-plus/techspec/), Expansion Slots, Storage, Note | หน้านี้แยก physical slot กับ mode, ให้ข้อมูลต่างตาม CPU และมีเงื่อนไข M.2/SATA disable | ต้องมี CPU-specific profile และ sharing rules ไม่รวม port counts ดิบ; ไม่มีการคัดลอก rule ของบอร์ดนี้ไปยัง AM5 fixture |
| S3 | [NVIDIA RTX 4090 Quick Start Guide](https://www.nvidia.com/content/geforce-gtx/GeForce_RTX_4090_QSG_Rev1.pdf), English System Specifications และ Hardware Installation | คู่มือแยก motherboard graphics slot, card clearance, system PSU และทางเลือก power connectors | แยก GPU-MB, GPU-PSU และ GPU-Case เป็นคนละ check ไม่สรุปว่าผ่านทั้งระบบจาก slot check |
| S4 | [NVIDIA Compare Graphics Cards](https://www.nvidia.com/en-us/geforce/graphics-cards/compare/), note ใต้ specifications | ตาราง reference/Founders Edition อาจต่างจากการ์ดของ add-in-card manufacturer และกำลังระบบมี configuration context | ต้องใช้หลักฐานตรง SKU/board variant/revision และแยก board_power_w จาก min_system_power_w |
| S5 | [Seasonic 12V-2x6 Cable](https://seasonic.com/seasonic-12v-2x6-cable/), requirements, compatible/not compatible series และ FAQ | การใช้สายขึ้นกับ PSU รุ่น/series และ connector/cable ที่กำหนด ไม่ใช่ดูรูปหัวหรือ wattage อย่างเดียว | power options เป็นชุดที่ผู้ผลิตรับรอง; ขาด cable identity/context ต้อง unknown ไม่อนุญาต adapter ทั่วไปโดยเดา |
| S6 | [Fractal North XL radiator compatibility](https://support.fractal-design.com/support/solutions/articles/4000203863), Things to keep in mind | ข้อจำกัด GPU เปลี่ยนตาม radiator/side bracket และการติดตั้ง drive trays | Case ต้องมี clearance profiles และ installation context ไม่รับรอง fit จาก max GPU length ค่าเดียว |
| S7 | [MSI PCIe Lane Bifurcation and Bandwidth Sharing](https://www.msi.com/blog/pcie-lane-bifurcation-and-bandwidth-sharing-on-msi-x870e-x870-motherboards), slot tables/notes/BIOS configuration | slots/M.2/USB4 อาจใช้ lanes ร่วมกันและ configuration มีผล | allocator ต้องตรวจร่วมหลายอุปกรณ์; ไม่ทราบ peripheral/BIOS context อาจต้อง unknown |

คำอธิบายทั้งหมดเป็นการสรุป ไม่คัดลอกคู่มือหรือภาพ ตาราง fields/API/limits/status aggregation
เป็นข้อเสนอออกแบบของ SpecUse ที่อนุมานจากความจำเป็นในการแทนเงื่อนไขเหล่านี้ ไม่ใช่กฎ API จากผู้ผลิต

## หลักฐานที่ต้องเก็บก่อนให้ check เป็น passed/failed

- SKU/revision จริงของสินค้า ไม่ใช่แค่ชื่อ chipset หรือ socket
- URL ผู้ผลิต, revision/วันที่เอกสารเมื่อมี, section/page ที่รองรับ field/rule และวันที่ตรวจ
- CPU configuration, slot identity, conditions และ scope ที่ข้อมูลครอบคลุม
- ถ้าเอกสารไม่ระบุหรือแหล่งข้อมูลขัดกัน ให้ null/unknown และเปิด data-review decision
- เมื่อเอกสารอัปเดต ให้ version repository snapshot/rule evidence และ revalidate ไม่ใช้ผลเก่าเป็นความจริงถาวร

## สิ่งที่เอกสารเหล่านี้ไม่รองรับให้อนุมาน

- ไม่รองรับการกำหนด specs ให้ GPU/Storage ที่ยังไม่มี SKU ใน catalog
- ไม่ยืนยันว่า M.2 ทุกช่องเป็น NVMe, PCIe x16 ทางกายภาพมี lanes x16 เสมอ หรือ ports ทุกช่องใช้พร้อมกันได้
- ไม่ยืนยันว่า VRAM/drive capacity/ราคามากแปลว่าต้องใช้ PSU รุ่นใด
- ไม่ให้สูตร power headroom สากล หรือสิทธิ์ใช้สาย/adapter ข้ามรุ่นโดยไม่ตรวจคู่มือ
- ไม่ยืนยัน boot/OS/driver, display path หรือความครบของ build จาก socket/RAM type ที่ตรงกัน
