# ข้อเสนอ API: GPU และ Storage — รอ Fluke ตรวจ

สถานะ **proposal เท่านั้น ไม่ใช่ API ที่เปิดใช้แล้ว** วันที่ตรวจ 2026-11-11
Branch `docs/gpu-storage-api-contract` จาก `origin/main` commit
`642bdab327adfd9761f9debfa05b52fd196df9fa` ไม่มีการแก้ Go/SQL/Frontend/runtime fixture
ชื่อ fields, enums, limits และ mode ใหม่ด้านล่างต้องตกลงร่วมกันก่อน implementation
รอบแก้ตาม review PR #3 เพิ่มเฉพาะเอกสารและ synthetic JSON สำหรับตกลง contract
ไม่ใช่การเปิด API หรือเพิ่มสินค้าเข้า repository ที่ server ใช้งาน

## 1. สิ่งที่ตรวจจาก main

- `internal/catalog/product.go`: คง 6 fields `id,name,category,brand,price_thb,is_active`;
  specs มีเฉพาะ cpu_socket, ram_type, modules_per_package และ RAM ram_model_id ระดับบน
- `internal/repository/build_memory.go`: ใช้ immutable snapshot ร่วมกับ validate;
  หมวดที่ไม่ใช่ CPU/motherboard/RAM ยังไม่อนุญาต typed specs และ GET ไม่คืน specs ของหมวดนั้น
- `internal/build/service.go`: non-RAM ใช้ items อยู่แล้ว คิดราคา
  `(quantity-owned_quantity)*catalog.price_thb`; ปฏิเสธ product/model IDs ซ้ำ; RAM ใช้ ram_items
- `internal/build/compatibility.go`: ตรวจเพียง CPU socket/RAM type กับ motherboard;
  GPU/Storage ไม่มี hardware checks จึงห้ามแปล compatible=true ว่าตรวจ GPU/Storage ผ่าน
- `internal/handler/build.go`: request รับเฉพาะ items/ram_items; unknown body fields เป็น 400;
  compatible เป็น bool, issues เป็น `{code,message}[]`, incompatibility ตอบ 200
- `frontend/src/pages/BuildPage.jsx`: payload ปัจจุบันมี CPU/motherboard/optional RAM;
  รอ CPU+motherboardก่อนเรียก validate; ตรวจว่า compatible เป็น boolean และบางตำแหน่งแสดง
  “เข้ากันได้ทั้งหมด” จาก boolean นี้ ส่วน API client เรียก path `/builds/validate` ตรง ๆ
  การอ่านไฟล์ FE ครั้งนี้ใช้ประเมินการย้ายสัญญาเท่านั้น ไม่แก้ไฟล์ FE
- FE มี category GPU/Storage อยู่ในรายการแล้ว แต่ VALIDATED_CATEGORIES ยังมี CPU/motherboard/RAM
  และ payload ยังไม่ส่ง GPU/Storage; ตัวอ่าน Catalog ยอมรับ category อื่นเมื่อ 6 fields เดิมถูกต้อง
  จึงเพิ่ม Catalog fields แบบ additive ได้ แต่ยังต้องให้ Fluke เพิ่ม selection/counts และ status UI
  ก่อนเปิดใช้งาน GPU/Storage จริง ไม่ใช่เพียงเพิ่มสินค้าแล้วคิดว่าหน้าปัจจุบันส่งข้อมูลครบ
- Backend บน main ตรงกับ `b65e6ac`; main มีการเปลี่ยน spec/Frontend เพิ่มเติมแล้ว
  ไม่มี GPU/Storage SKU ใน local fixture จึงยังไม่มี product specs จริงให้แปลงหรือ backfill

## 2. หลักของสัญญา

คง `/api/v1`, envelopes/pagination, UUID, category ตัวพิมพ์เล็ก, `is_active` และราคาเงินบาทจำนวนเต็ม
ทั้ง GET `/products` และ GET `/products/{id}` ส่ง specs แบบเดียวกันจาก repository เดียวกับ validate
เพิ่ม category `gpu`/`storage` โดยไม่สร้าง `gpu_items` หรือ `storage_items`; `ram_items` คงเดิม
ไม่เพิ่ม filter เฉพาะฮาร์ดแวร์ในรอบแรก ใช้ category/search/page/page_size เดิม

**Display ไม่ใช่ compatibility evidence:** `specs.display` ใช้แสดง/ค้นเท่านั้น;
`specs.host`, `power`, `dimensions`, motherboard connectivity ใช้ตัดสินได้เมื่อมีหลักฐานตรง SKU/revision
ห้าม infer จากชื่อสินค้า, chipset name, ราคา, VRAM/capacity, คำว่า “gaming”, หรือผลที่ client คำนวณ

ข้อมูลที่ยังไม่ทราบใช้ `null` ไม่ใช้ 0/false/[] แทน unknown:

- scalar/array/object ใหม่ที่ยังไม่มีข้อมูลส่ง null; legacy fields ที่มีแล้วไม่เปลี่ยนเป็น nullable
- [] หมายถึงตรวจแล้วว่าไม่มีสมาชิก/ไม่มี rule ใน scope ที่ระบุ ไม่ใช่ยังไม่อ่านคู่มือ
- `other` หมายถึงทราบว่าอยู่นอก enum ที่รองรับ ไม่ใช่ไม่ทราบ (กรณีนั้นใช้ null)
- ไม่คัดลอก specs การ์ด reference ไปใช้กับ SKU ของผู้ผลิตบอร์ดรายอื่น [S4](https://www.nvidia.com/en-us/geforce/graphics-cards/compare/)
- field names/limits ใน proposal เป็นการออกแบบของ SpecUse ไม่ใช่ชื่อมาตรฐานที่ผู้ผลิตกำหนด

## 3. GET specs ที่เสนอ

`?` ในตารางหมายถึง nullable; fields ทั้งหมดอยู่ใน specs นอกจากที่ระบุ ไม่บังคับ GPU/Storage
ที่ metadata ไม่ครบให้หายจาก Catalog: แสดงและคิดราคาได้ แต่ผลตรวจส่วนนั้นเป็น unknown

### 3.1 GPU

| Field | ชนิด/หน่วย/enum | ใช้เพื่อ |
|---|---|---|
| `display.chipset_name` | string? ชื่อจากผู้ผลิต | แสดงผลเท่านั้น ไม่เท่ากับ SKU/revision |
| `display.vram_capacity_bytes` | integer? >0, bytes | แสดงผล ไม่เป็นเกณฑ์ slot/PSU |
| `host.connector` | string? `pcie_x16`, `other` | physical edge connector จากคู่มือ SKU [S3](https://www.nvidia.com/content/geforce-gtx/GeForce_RTX_4090_QSG_Rev1.pdf) |
| `host.supported_link_modes` | LinkMode[]? | ความสามารถที่มีหลักฐาน ไม่ขยายจากค่า maximum เอง |
| `power.board_power_w` | number? >0, W | กำลังการ์ด ใช้ช่วง C ไม่เท่ากับกำลังทั้งระบบ |
| `power.min_system_power_w` | integer? >0, W | คำแนะนำ PSU ของผู้ผลิตภายใต้เงื่อนไขระบบอ้างอิง ไม่ใช่ load ของ GPU [S3](https://www.nvidia.com/content/geforce-gtx/GeForce_RTX_4090_QSG_Rev1.pdf), [S4](https://www.nvidia.com/en-us/geforce/graphics-cards/compare/) |
| `power.connector_options` | PowerOption[]? | ทางเลือกสาย native/adapter ที่ผู้ผลิตรับรอง ใช้ช่วง C |
| `dimensions.length_mm` | number? >0, mm | ความยาวการ์ดตามแกนที่ระบุในแบบผู้ผลิต |
| `dimensions.height_mm` | number? >0, mm | ความสูงเมื่อเสียบการ์ด ไม่เดาจากการเรียง L/W/H |
| `dimensions.thickness_mm` | number? >0, mm | ความหนาการ์ด ใช้กับ case/layout |
| `dimensions.occupied_slots` | integer? >=1, slots | จำนวนตำแหน่ง bracket ที่ต้องใช้ ไม่ปัดจาก mm เอง |

`LinkMode = {pcie_generation: integer>=1, lanes: 1|2|4|8|16}` ไม่มีหน่วย
เป็นชุดโหมดที่ผู้ผลิตยืนยันว่าใช้งานได้ ไม่ใช่การอ้างว่ารองรับทุก generation/lane ที่ต่ำกว่า
ถ้าทราบแค่ maximum แต่ไม่พอสรุปโหมดร่วม ให้ใช้ null และแจ้ง unknown แทนการเดา

`PowerOption = {option_id: string, adapter_part_number: string|null,
connectors: [{type: PowerConnector, count: integer>=1}],
cable_policy: independent_cables|manufacturer_cable_kit|unspecified}`
options เป็น OR; connectors ภายใน option เป็น AND; adapter_part_number=null หมายถึง
ทางเลือก native ที่ระบุเช่นนั้นในหลักฐาน ไม่ใช่อนุญาต adapter ใดก็ได้
`PowerConnector` enum: `pcie_6pin`, `pcie_8pin`, `12vhpwr`, `12v_2x6`, `sata_power`, `none`, `other`
`none` ใช้ count=0 เป็นกรณียกเว้นเมื่อเอกสารยืนยันว่าไม่ใช้สายเสริม; ไม่อนุมานจาก wattage
รูปร่าง connector เหมือนกันไม่พอรับรอง PSU-side pinout/cable compatibility [S5](https://seasonic.com/seasonic-12v-2x6-cable/)

### 3.2 Storage

| Field | ชนิด/หน่วย/enum | ใช้เพื่อ |
|---|---|---|
| `display.media_type` | string? `ssd`, `hdd`, `other` | แสดงผล ไม่ใช้เดา interface |
| `display.capacity_bytes` | integer? >0, bytes | ความจุสำหรับแสดงผล ไม่ใช้ตัดสิน motherboard |
| `display.sequential_read_mb_s` | number? >=0, decimal MB/s (10^6 bytes/s) | แสดงผล ไม่รับรองความเร็วจริงใน build |
| `display.sequential_write_mb_s` | number? >=0, decimal MB/s | แสดงผล |
| `host.connection_type` | string? `sata`, `pcie_nvme`, `pcie_other`, `sas`, `usb`, `other` | รวมชนิด transport/protocol ที่ตรวจยืนยันแล้ว ไม่เอา M.2 มาใช้แทน [S1](https://www.kingston.com/en/ssd/ssd-faq) |
| `host.connector` | string? `sata_data`, `m2`, `pcie_edge`, `u2`, `usb`, `other` | หัวต่อ ไม่ใช่ protocol |
| `host.form_factor` | string? `m2`, `2_5_inch`, `3_5_inch`, `pcie_aic`, `u2`, `other` | รูปทรง/การติดตั้ง ไม่อนุมานจากความจุ |
| `host.m2_keying` | string? `b`, `m`, `b_m` | ใช้เฉพาะ M.2 ร่วมกับ protocol/ขนาด ไม่ใช้ key เดี่ยว ๆ สรุปว่าใช้ได้ [S1](https://www.kingston.com/en/ssd/ssd-faq) |
| `host.width_mm`, `host.length_mm` | integer? >0, mm | ขนาด module M.2; ถ้าไม่ใช่ M.2 ใช้ null และไม่นำไปตรวจ M.2 |
| `host.supported_link_modes` | LinkMode[]? | ใช้เฉพาะ PCIe; generation/lane ต่ำกว่าไม่ถูกตัดสิน fail อัตโนมัติ [S1](https://www.kingston.com/en/ssd/ssd-faq) |
| `power.connector_type` | PowerConnector? | ช่วง C; `none` เมื่อไม่มีสายไฟแยกตามหลักฐาน ไม่แปลว่าใช้พลังงาน 0 |
| `power.max_operating_power_w`, `power.startup_power_w` | number? >=0, W | ช่วง C รวม startup ของอุปกรณ์ที่เกี่ยวข้อง; ห้ามแทน unknown ด้วย 0 |
| `dimensions.width_mm`, `dimensions.length_mm`, `dimensions.height_mm` | number? >0, mm | ขนาดตัว drive/ชุด heatsink ที่ขายจริงสำหรับตรวจ bay/layout ในช่วง C |

ขอบเขตตรวจช่วง B: SATA ภายในแบบ 2.5/3.5 นิ้ว และ M.2 SATA/PCIe NVMe เท่านั้น
SAS/U.2/AIC/USB/RAID/adapters อาจแสดงใน Catalog ได้ แต่การตรวจที่ยังไม่รองรับต้องเป็น
unknown/UNSUPPORTED_DEVICE_CLASS ไม่ใช่ hardware fail หรือคาดเดาว่าใช้งานได้
ข้อมูลความจุเป็น bytes ไม่แปลง GB เป็น GiB โดยไม่รู้หน่วยต้นฉบับ ถ้าหลักฐานไม่ชัดให้ null

### 3.3 Motherboard

คง `specs.cpu_socket` และ `specs.ram_type` เดิม เพิ่ม `specs.connectivity_profiles: IOProfile[]|null`
แทนการใช้เพียง gpu_slot_count/sata_count/m2_count ซึ่งไม่บอกทรัพยากรที่ใช้ร่วมกัน [S2](https://www.asus.com/motherboards-components/motherboards/prime/prime-b550-plus/techspec/), [S7](https://www.msi.com/blog/pcie-lane-bifurcation-and-bandwidth-sharing-on-msi-x870e-x870-motherboards)

| IOProfile field | ชนิด/หน่วย/ความหมาย |
|---|---|
| `profile_id` | string, ID เฉพาะภายใน motherboard SKU ไม่ใช่ UUID ของสินค้า |
| `cpu_product_ids` | UUID[] ไม่ว่าง หรือ null เมื่อยังไม่รู้; CPU catalog ที่หลักฐานยืนยันว่า profile นี้ใช้ได้ |
| `requires_firmware_confirmation` | boolean?; true/unknown โดยไม่มี BIOS context ทำให้ checks ที่เกี่ยวข้องเป็น unknown |
| `pcie_slots` | PcieSlot[]?; [] คือยืนยันว่าไม่มี slot ที่เข้า scope |
| `storage_ports` | StoragePort[]?; [] คือยืนยันว่าไม่มี port ที่เข้า scope |
| `resource_rules` | ResourceRule[]?; null คือยังไม่ทราบ sharing rules, [] คือ review ว่าไม่มี rules ใน scope นี้แล้ว |

**PcieSlot:** `slot_id:string`, `connector:pcie_x1|pcie_x4|pcie_x8|pcie_x16|null`,
`supported_link_modes:LinkMode[]|null`, `gpu_supported:boolean|null`
physical x16 ไม่ได้แปลว่า wiring x16; ต้องดู profile ของ CPU ที่เลือก [S2](https://www.asus.com/motherboards-components/motherboards/prime/prime-b550-plus/techspec/)

**StoragePort:** `port_id:string`, `connector:sata_data|m2|other|null`,
`supported_connection_types:ConnectionType[]|null` (enum เดียวกับ Storage),
`accepted_module_keys:(b|m|b_m)[]|null`, `supported_widths_mm:integer[]|null`,
`supported_lengths_mm:integer[]|null`, `supported_link_modes:LinkMode[]|null`
keys/ขนาด/link modes ที่ไม่เกี่ยวข้องกับ port นั้นใช้ null โดย checker ข้ามเฉพาะ dimension
ที่ยืนยันว่าไม่ applicable ไม่ถือว่า port ทั้งหมดมีข้อมูลครบจาก null เหล่านั้น

**ResourceRule:** `rule_id:string`, `when_all_occupied:string[]` (slot/port IDs ใน profile),
`effect:disable|restrict_link_modes|requires_external_context`, `affected_endpoint_ids:string[]`,
`allowed_link_modes:LinkMode[]|null`, `required_context:string[]`, `source_ids:string[]` ไม่ว่าง
disable ไม่ใช้ allowed_link_modes; restrict_link_modes ต้องมีรายการยืนยัน; external context
เช่นการใช้ USB4/BIOS ที่ request ยังไม่บอกต้องทำให้ผลที่ขึ้นกับเงื่อนไขนั้นเป็น unknown [S7](https://www.msi.com/blog/pcie-lane-bifurcation-and-bandwidth-sharing-on-msi-x870e-x870-motherboards)
when_all_occupied และ affected_endpoint_ids ต้องไม่ว่างและอ้าง IDs ใน profile ที่มีจริง
required_context ใช้ [] สำหรับ disable/restrict; requires_external_context ต้องมีอย่างน้อยหนึ่งค่า
จาก `bios_version`, `usb4_usage`, `unlisted_slot_occupancy` และยังไม่สามารถรับค่าเหล่านี้ใน request รุ่นนี้
เช่น rule ที่ trigger เมื่อ M.2 ถูกเลือกและต้องรู้การใช้ USB4 จะชี้ M.2 ใน when_all_occupied
กับ required_context=["usb4_usage"] ไม่อ้าง USB4 เป็น slot/port สมมติที่ไม่อยู่ใน profile

เลือก IOProfile จาก CPU UUID ที่อยู่ใน items; ไม่มี/หลาย motherboard หรือไม่มี CPU ที่เลือก
profile ได้แน่นอน => unknown สำหรับ checks ที่เกี่ยวข้อง ไม่เลือกรุ่น CPU จาก socket อย่างเดียว
ถ้ามีหลาย profile ที่เป็นไปได้ ให้ pass เฉพาะเมื่อพิสูจน์ผลเดียวกันได้ทุก profile มิฉะนั้น unknown
ไม่ถือว่า cpu_product_ids=[] เป็น wildcard

### 3.4 หลักฐานของแต่ละ SKU และความครบของข้อมูล

เสนอ `specs.sources: Source[]|null`, `specs.evidence: Evidence[]|null` สำหรับ GPU/Storage/motherboard
`Source={source_id:string, manufacturer:string, title:string, url:string(https),
document_revision:string|null, section:string, checked_at:string(RFC3339)}`
`Evidence={paths:string[](JSON Pointer ภายใน specs), source_ids:string[]}`
ข้อมูลที่ใช้ตัดสินต้องมี source ตรง SKU/revision/conditions และผ่านการ review ฝั่ง Backend
client ส่ง evidence/specs/ผลตรวจมาแทนไม่ได้ ข้อมูล display อย่างเดียวไม่ทำให้ checks ผ่าน
links ในเอกสารนี้เป็นหลักฐานออกแบบกฎ ไม่ได้ผูก specs ของสินค้าตัวอย่างเข้ากับรุ่นผู้ผลิตนั้น

**ข้อยกเว้นเฉพาะตัวอย่างทดสอบ:** ชุด synthetic ใช้ค่าที่แต่งขึ้นอย่างเปิดเผยเพื่อทดสอบ schema/สถานะ
ไม่ใช่ข้อมูลสินค้าจริง ไม่ใช้ชื่อ/UUID ของสินค้า runtime แหล่ง `synthetic-v1` และ URL โดเมน `.invalid`
เป็นเครื่องหมายข้อมูลจำลอง ไม่ใช่หลักฐานผู้ผลิต ใช้เฉพาะ contract test/mock ที่ตั้งค่าแยก;
production ต้องปฏิเสธหลักฐาน synthetic นี้ ดู [คู่มือตัวอย่าง](examples/README.md)

## 4. Request, จำนวนอุปกรณ์ และ owned

โครงสร้าง body **เดิมเท่านั้น**: `items:[{product_id,quantity,owned_quantity}]`,
`ram_items:[{ram_model_id,required_modules,owned_modules}]`; ต้องมีทั้งสอง arrays ใช้ [] ได้
RAM ไม่ถูกแปลงมาเป็น product items; ไม่มี client price, slot choice, specs หรือผลตรวจใน request

ข้อเสนอขอบเขต software สำหรับโหมดใหม่ (ต้องให้ Fluke ยืนยัน ไม่ใช่ข้อจำกัดทางฮาร์ดแวร์):

| หมวด | จำนวนรวมที่ต้องการติดตั้ง รวมของที่มีแล้ว | กฎ |
|---|---|---|
| GPU แยก | 0–1 ใบ | ไม่เลือกได้; ถ้าเลือก quantity=1, owned_quantity=0/1; multi-GPU อยู่นอก scope |
| Storage ทุกชนิด | 0–8 ลูก รวมทุก SKU/category=storage | รวมชนิดที่ checker ยังไม่รองรับ เช่น USB/SAS/other; ไม่สร้าง row product_id ซ้ำ |

ทุก row quantity>=1, 0<=owned_quantity<=quantity จำนวนรวมคิด SUM(quantity) ตาม category
ไม่ใช่จำนวนแถวและไม่ใช่ SUM(quantity-owned_quantity); owned คือส่วนหนึ่งของ quantity
Storage quantity=3, owned=2 ต้องมีที่ติดตั้ง/port/ไฟสำหรับ 3 ลูก แต่จ่ายเพิ่ม 1 ลูก
GPU owned=1 ยังใช้ slot/PSU/Case เหมือน GPU ที่ซื้อใหม่ ถึง purchase_price_thb=0
ของที่ไม่ได้อยู่ใน build ไม่ต้องส่ง row; อย่าส่ง quantity=0 แทนการไม่เลือก

**8 ลูกเป็น software scope เท่านั้น** ไม่รับรองว่า motherboard มี 8 พอร์ตหรือใช้พร้อมกันได้
resolve category จาก catalog ฝั่ง server แล้วนับ `SUM(quantity WHERE category == "storage")`
รวมชนิดที่ยังตรวจไม่ได้/metadata ไม่ครบ/owned ห้ามนับเฉพาะ SATA/M.2 หัก owned หรือบวก owned ซ้ำ

ลำดับ: ตรวจรูปแบบ/จำนวน/duplicates ตามเดิม → resolve product และ RAM model ทั้งหมดตาม policy เดิม
→ ถ้าพบไม่พบ/inactive ให้ 404 `NOT_FOUND` โดยไม่เดา category หรือข้ามแถวนั้น
→ จึงตรวจ software limit → ราคา/compatibility; ไม่ใช้ unsupported class หลบ limit
เช่นมี USB quantity=9 และอีก ID ไม่พบ ต้อง 404 ก่อน ไม่ตอบ scope error จากข้อมูลที่ resolve ไม่ครบ

| รายการ Storage | จำนวนติดตั้ง/นับ limit | ซื้อเพิ่ม | ผลเฉพาะ software limit |
|---|---:|---:|---|
| SATA quantity=3, owned=2 | 3 | 1 | ผ่าน; ไม่ใช่ 5 และไม่ใช่ 1 |
| SATA 3/owned 2 + NVMe 2/owned 2 + USB 3/owned 1 | 8 | 3 | ผ่าน แม้ USB check ยัง unknown |
| SATA 4/owned 4 + NVMe 2/owned 2 + USB 3/owned 3 | 9 | 0 | 400 UNSUPPORTED_BUILD_SCOPE |
| USB quantity=9, owned=0 | 9 | 9 | 400 UNSUPPORTED_BUILD_SCOPE ก่อน hardware check |

ตัวอย่าง: [storage-counting.scenarios.proposed.json](examples/storage-counting.scenarios.proposed.json)

ไม่เปลี่ยน policy ปัจจุบันที่ inactive product อ้างอิงไม่ได้ (404 แม้ owned) ในงานนี้
การย้าย build เก่าหรือการอ้างอิงของ owned ที่ deactivate ต้องตกลงแยก ไม่สร้างข้อยกเว้นเงียบ ๆ
ราคาเป็น current catalog price, หัก owned เฉพาะเงิน, คำนวณ RAM แบบเดิม และตรวจ overflow
การไม่มี discrete GPU ไม่พิสูจน์ว่า CPU มี iGPU/ภาพออกจอได้; display path ยังอยู่นอก checks

## 5. Passed / failed / unknown โดยไม่ทำ client เดิมพัง

**ไม่ใช้ compatible=null/string และไม่เปลี่ยน issues เป็น object** เพราะ FE บน main ตรวจชนิดอยู่
การเพิ่ม status อย่างเดียวแต่ยังให้ UI เดิมอ่าน bool ไม่แก้ปัญหา: compatible=true อาจแสดงว่า
“เข้ากันได้ทั้งหมด”; compatible=false สำหรับ unknown อาจแสดงว่า “ไม่เข้ากัน”
จึงไม่สามารถรับรอง UI ที่ถูกต้องด้วยการแก้ Backend แบบ additive เพียงฝั่งเดียว

### ข้อเสนอที่แนะนำ: opt-in โหมดใหม่ ใช้ body เดิม

- Default `POST /api/v1/builds/validate` คงสัญญา/ความหมาย bool เดิม: checks CPU socket/RAM type เท่านั้น
  รวมพฤติกรรม issues ของ metadata เดิม ไม่เพิ่ม compatibility object หรือเปลี่ยน bool/default limits เงียบ ๆ
  legacy compatible=true อาจหมายถึง CPU/RAM ผ่าน แต่ยังไม่เคยตรวจ GPU/Storage/PSU/Case
- เสนอ `POST /api/v1/builds/validate?validation_mode=gpu_storage_v1` สำหรับ FE ที่รองรับ tri-state แล้ว
- โหมดใหม่คง fields เดิมและเพิ่ม `data.compatibility` ตามตารางด้านล่าง
- ในโหมดใหม่ `compatible = (compatibility.status == "passed")`: unknown => false แต่ issues ว่างได้
  **false ไม่ใช่ข้อสรุปว่า incompatible** ในโหมดนี้ ต้องอ่าน status; legacy endpoint ไม่เปลี่ยน semantics
- FE ต้องยืนยัน `compatibility.mode==gpu_storage_v1` ก่อนแสดงผล/เชื่อว่าเป็นการตรวจใหม่
  Server ปัจจุบันยังไม่ implement mode และอาจ ignore query แล้วคืน legacy body จึงห้าม fallback
  ไปตีความ bool เป็นผลใหม่ถ้า compatibility object หาย
- ไม่เปิด GPU/Storage validation UI แบบใหม่จน FE รองรับ status/coverage; คุย migration ร่วมกันก่อน
  รอบนี้ไม่แก้ FE และไม่เปลี่ยน default path/ผล CPU+RAM ที่ฟลุคใช้อยู่
- หากไม่ตกลง opt-in ให้พิจารณา version ใหม่แทน ห้าม rollout เปลี่ยน default bool โดยปริยาย

| `data.compatibility` field | ชนิด/กฎ |
|---|---|
| `mode` | string คงที่ `gpu_storage_v1` สำหรับ proposal นี้ |
| `status` | `passed`, `failed`, `unknown` ตาม checks ที่เกี่ยวข้องทั้งหมด รวม PSU/Case |
| `checks` | Check[] ตาม required-check rules ด้านล่าง; ห้ามละ required check เพราะข้อมูลขาดหรือยังไม่ implement |
| `warnings` | `{code,message}[]` เช่นลด bandwidth ที่เอกสารยอมรับ ไม่ปนกับ proven-failure issues |
| `unverified_domains` | string[] จาก scope `basic`, `motherboard`, `psu`, `case` ที่ยังมี unknown; ไม่ซ้ำและเรียงชื่อ; overall failed ก็มีรายการนี้ได้ |
| `build_completeness` | คงที่ `not_evaluated` จนมีสัญญาเกณฑ์ความครบ; ไม่ใช่สิทธิ์ publish |

`Check={code:CheckCode, scope:basic|motherboard|psu|case,
status:passed|failed|unknown, reason_code:string, message:string,
product_ids:UUID[], missing_fields:FieldReference[], unsupported_fields:FieldReference[]}`
ทั้งสอง arrays ต้องมีในทุก check และใช้ [] เมื่อไม่มี ห้ามสลับเป็น string[]/null/ละ field ตามสถานะ
`unsupported_fields` เพิ่มใน proposal รอบนี้เพื่อแยกข้อมูลที่มีแต่ตรวจไม่รองรับจากข้อมูลขาด
ไม่เพิ่ม arrays เหล่านี้ใน legacy issues หรือ error envelope
CheckCode: `CPU_SOCKET`, `RAM_TYPE`, `GPU_MOTHERBOARD`, `STORAGE_MOTHERBOARD`,
`MOTHERBOARD_RESOURCES`, `GPU_PSU`, `GPU_CASE`, `STORAGE_PSU`, `STORAGE_CASE`, `SYSTEM_PSU`, `SYSTEM_CASE`

Required checks กำหนดจากสิ่งที่เลือก ไม่ใช่จากสิ่งที่ implementation ทำได้:

| Check | เมื่อจำเป็น |
|---|---|
| CPU_SOCKET | เลือกทั้ง CPU และ motherboard; ตาม legacy rule ไม่ใช่ CPU support/BIOS ทั้งหมด |
| RAM_TYPE | มี ram_items; ยังไม่เลือก motherboard ให้ unknown พร้อม reference อุปกรณ์ที่ขาด |
| GPU_MOTHERBOARD | มี GPU แม้ยังไม่มี motherboard/CPU สำหรับเลือก IOProfile |
| STORAGE_MOTHERBOARD | มี Storage ทุกชนิด แม้ชนิดนั้นยังไม่รองรับ |
| MOTHERBOARD_RESOURCES | มี GPU หรือ Storage; รวมทุก instance รวม owned |
| GPU_PSU / GPU_CASE | มี GPU |
| STORAGE_PSU / STORAGE_CASE | มี Storage |
| SYSTEM_PSU / SYSTEM_CASE | มี items หรือ ram_items อย่างน้อยหนึ่งรายการ |

ไม่เลือก GPU/Storage ให้ละ checks เฉพาะหมวดนั้นได้ แต่เลือกแล้วขาด motherboard ต้องส่ง unknown
ห้ามละ check จน aggregate เป็น passed; product_ids มี RAM SKU ที่เลือก/อ้างอิงจริง ไม่ใช่ model ID
กรณี RAM owned ทั้งหมดและไม่มี SKU อ้างอิง product_ids อาจมีเพียง motherboard; ไม่ปลอม model เป็น product

เมื่อมีอุปกรณ์ใดถูกเลือกในโหมดใหม่ ต้องมี SYSTEM_PSU/SYSTEM_CASE เป็น required checks ด้วย
แม้ไม่เลือก discrete GPU/Storage เพื่อไม่ให้อุปกรณ์ประเภทอื่นที่ยังไม่ตรวจไฟ/พื้นที่หลุดจาก coverage
ช่วง A/B สอง checks นี้เป็น unknown/NOT_IMPLEMENTED จึงไม่ให้ overall passed โดยไม่มี PSU/Case
นี่เป็น policy ความครอบคลุมของโหมดใหม่ ไม่เพิ่มหมวดบังคับให้ legacy หรือใช้ตัดสินความครบของ build

- passed: มีข้อมูลและหลักฐานพอสำหรับ check นั้น และพิสูจน์ว่าเงื่อนไขที่ประกาศผ่าน
- failed: มีหลักฐานพอที่จะพิสูจน์ conflict จริง ไม่ใช้ความไม่รู้เป็น fail
- unknown: ยังไม่มี implementation/data/context หรือ hardware class อยู่นอก scope
- aggregate **ตามลำดับนี้เสมอ**: (1) มี required check failed ที่ยืนยันได้อย่างน้อยหนึ่ง → failed
  แม้มี unknown/PSU/Case ยังไม่ตรวจ; (2) ไม่มี failed แต่มี required check unknown → unknown;
  (3) required checks ไม่ว่างและผ่านทุกอัน → passed; (4) ไม่มี check เลย → unknown ไม่ใช่ vacuous pass
- `issues` เดิมใส่เฉพาะ failures เป็น `{code,message}`; unknown อยู่ใน checks/missing_fields ไม่ทำให้เป็น server error
- ช่วง A/B **ไม่ใช่ unknown ทุกกรณี**: ถ้า CPU/RAM หรือ motherboard check ที่ทำแล้ว fail → overall failed
  ถ้า checks ที่ทำได้ผ่านและยังมี PSU/Case unknown → overall unknown; ยังไม่สามารถ overall passed
- ระบบที่เลือกเพียง CPU/RAM ก็ยังต้องอ่าน coverage ของโหมดใหม่ ไม่ใช้การละ GPU/Storage หลบ SYSTEM checks
- แม้ช่วง C ได้ passed หมายถึงผ่านเฉพาะ checks ที่ประกาศ ไม่รวม boot/OS/driver/performance/การประกอบจริง
  UI ใช้ “ผ่านการตรวจที่ระบุ” ไม่ใช้ “ตรวจครบทุกด้าน/พร้อมเผยแพร่”

| Required checks (ตัวอย่าง) | Overall | compatible ใน opt-in | ความหมาย |
|---|---|---|---|
| passed + passed + PSU unknown + Case unknown | unknown | false | ผ่านบางส่วน ยังยืนยันที่เหลือไม่ได้ |
| failed + passed + PSU unknown + Case unknown | failed | false | พบ conflict แน่นอนแม้ตรวจยังไม่ครบ |
| passed ทุก required check (อนาคต C ที่ข้อมูล/implementation ครบ) | passed | true | ผ่านเฉพาะ checks ที่ประกาศ |
| [] | unknown | false | ไม่มีผลให้ยืนยัน |

ดู [overall-status.scenarios.proposed.json](examples/overall-status.scenarios.proposed.json)
ค่า bool ใน opt-in เท่านั้นที่เท่ากับ overall==passed; ห้ามนำ rule นี้ไปเปลี่ยน legacy client

### 5.1 FieldReference: missing กับ unsupported

`FieldReference={location:catalog|request|context, path:string(JSON Pointer),
product_id:UUID|null, category:string|null}` — ต้องส่ง keys ทั้ง 4 เสมอ ไม่มี value/specs จาก client

| location | path อ้าง root ใด | product_id / category |
|---|---|---|
| catalog | Product ของ ID นั้น เช่น `/specs/host/connector` | UUID สินค้าจริงใน snapshot + category |
| request | Request body เช่น `/items` หรือ `/ram_items` | ยังไม่เลือกอุปกรณ์ใช้ product_id=null + category ที่ขาด เช่น `psu` |
| context | บริบทที่เชื่อถือได้ซึ่งยังไม่ใช่ request fields เช่น `/installation_context` | ระบบรวมใช้ null/null; ถ้าผูกสินค้าระบุ UUID/category |

path มี leading `/` และใช้ escaping JSON Pointer (`~0`, `~1`); root catalog เป็น **Product** ไม่ใช่ specs
เมื่อไม่เลือก PSU ใช้ `{location:"request",path:"/items",product_id:null,category:"psu"}`
ไม่สร้าง UUID สมมติ/ใช้ empty string; product_ids ใน check มีเฉพาะ UUID สินค้าที่เลือกหรือ RAM SKU จริง
ห้ามอ้าง catalog ของสินค้าที่ไม่ได้เลือกแทน missing component; context path ไม่ใช่การอนุญาต body fields ใหม่

- `missing_fields`: อุปกรณ์/metadata/บริบทที่จำเป็นแต่ไม่มีหรือเป็น null ไม่ใช่รายการ checker ที่ยังไม่เขียน
- `unsupported_fields`: ทราบค่าแล้วแต่ class/feature อยู่นอกขอบเขต เช่น connection_type=`usb`
  → unknown/UNSUPPORTED_DEVICE_CLASS, missing_fields=[] และอ้าง `/specs/host/connection_type` ใน unsupported_fields
- null ที่ยืนยันว่าไม่ applicable เช่น M.2 key ของ SATA 2.5 นิ้ว ไม่อยู่ใน missing_fields
- NOT_IMPLEMENTED ไม่ใช่ missing data โดยตัวมันเอง ถ้าข้อมูลครบแต่ไม่มี checker ส่งทั้งสอง arrays=[];
  ถ้าไม่ได้เลือก PSU จริงให้รายงาน missing PSU ได้ด้วย
- มีทั้งข้อมูลขาดและ feature ไม่รองรับให้ส่งทั้งสอง arrays ตามจริง ห้ามย้ายค่าที่มีไปเป็น missing
- ผ่าน/ไม่ผ่านจากข้อมูลครบ: missing_fields=[], unsupported_fields=[]; failed ที่พิสูจน์ได้อาจมี
  missing ของเงื่อนไขอื่น แต่ความไม่รู้ไม่ลบล้าง conflict ที่ยืนยันได้
- เรียง reference ตาม location, product_id (null ก่อน), category (null ก่อน), path และตัดรายการซ้ำ
  ใช้รูปแบบนี้ในทุก full response/check fragment ไม่มี missing_fields ระดับ data หรือ legacy error เพิ่ม

ดูตัวอย่างข้อมูลขาด/อุปกรณ์ยังไม่เลือก/บริบทขาด/known unsupported/ไม่มีข้อมูลขาดใน
[คู่มือตัวอย่าง](examples/README.md) และ JSON ที่เชื่อมไว้

## 6. กฎ Motherboard ที่เสนอสำหรับช่วง B

ต้องพิจารณา GPU และ Storage ทุก instance รวม owned พร้อมกัน ใช้ deterministic search
หา allocation ที่ทำได้ก่อน ไม่ใช้ greedy จอง port แรกแล้วสรุปว่าไม่มีที่เหลือ
เรียง product UUID + instance ordinal และ endpoint IDs เพื่อให้ผลคงที่หลังพิสูจน์ feasibility
instance ordinal เป็นเลขภายใน row 1..quantity ไม่ใช่ product UUID ใหม่

1. GPU: ตรวจ physical connector/graphics support และโหมด link ที่มีหลักฐานร่วมกัน
   แยกขนาด connector จากจำนวน lanes ที่ wiring จริง; ไม่ถือว่า gen ต่างกันหรือ lanes ต่ำกว่า
   เป็น incompatibility อัตโนมัติ ถ้าเอกสารยืนยันว่าทำงานที่โหมดลดลงให้ผ่านพร้อม warning [S2](https://www.asus.com/motherboards-components/motherboards/prime/prime-b550-plus/techspec/), [S3](https://www.nvidia.com/content/geforce-gtx/GeForce_RTX_4090_QSG_Rev1.pdf)
2. Storage: ตรวจ connector + connection_type + M.2 key/width/length ตามที่ applicable
   M.2 ไม่ได้แปลว่า NVMe เสมอ และเสียบได้ทางกายภาพไม่พอรับรอง protocol [S1](https://www.kingston.com/en/ssd/ssd-faq)
3. จัดสรร storage 1 unit ต่อ port ที่ใช้ได้ และ GPU ต่อ slot พร้อมประเมิน disable/restrict rules
   ของ IOProfile ที่ตรง CPU; ไม่เอาจำนวน SATA+M.2 รวมดิบมาใช้แทนความจุที่ใช้พร้อมกันได้ [S2](https://www.asus.com/motherboards-components/motherboards/prime/prime-b550-plus/techspec/), [S7](https://www.msi.com/blog/pcie-lane-bifurcation-and-bandwidth-sharing-on-msi-x870e-x870-motherboards)
4. มี verified feasible assignment และข้อมูล constraints ที่เกี่ยวข้องครบ => ผ่าน motherboard resources
   ไม่มี assignment ภายใต้ข้อมูลที่ครบพอพิสูจน์ => failed; ยังมี candidate/rule/profile ที่ไม่รู้
   และอาจเปลี่ยนคำตอบ => unknown ไม่สรุป fail เพราะค้นเฉพาะข้อมูลที่รู้แล้วไม่เจอ
5. ข้อมูล PSU/Case ไม่ถูกตรวจในช่วงนี้ แยก check เป็น unknown เสมอ ไม่อนุมานจาก motherboard
   generation, ขนาด VRAM/drive หรือค่าพลังงานแนะนำเพียงค่าเดียว

การ allocation นี้ตอบว่า “จัดลงได้อย่างน้อยหนึ่งแบบ” ไม่ยืนยันการเสียบที่ผู้ใช้ทำอยู่จริง
ช่วงแรกไม่ส่ง slots back ใน request; การ lock port/boot drive/RAID/BIOS context เป็น proposal ถัดไป
USB4/add-in cards/อุปกรณ์นอก items ที่อาจใช้ resource ร่วม แต่ request ไม่บอก ต้องเป็น unknown

## 7. Responses และ error codes

GET ใช้ envelopes เดิม; unknown metadata ไม่ใช่ 404 และไม่ถูกแปลงเป็น specs ที่แต่งขึ้น
ตัวอย่างทั้งหมดใน [examples/](examples/) เป็น **เอกสารทดสอบ ไม่ใช่ runtime seed/สินค้าใหม่/ผลจาก API ปัจจุบัน**
GPU/Storage IDs และชื่อเป็นตัวอย่าง ราคา 0 เป็น placeholder ไม่ใช่ราคาจริง ทุก hardware field ใหม่
ใน Catalog examples **ชุดเดิม** เป็น null; motherboard ชุดเดิมใช้ fixture เดิมและ connectivity=null
ส่วนชุด synthetic ใหม่ใช้ UUID แยก ราคา/metadata ที่แต่งเพื่อทดสอบอย่างเปิดเผย
มี GPU, M.2 NVMe/SATA/USB, motherboard สองแบบ, CPU และ RAM ไม่ต้อง hardcode mapping
PSU/Case ในชุดนี้มีเพียง 6-field placeholders สำหรับตัวอย่างอนาคต ไม่ได้ตกลง specs ของหมวดนั้น

- [GET list](examples/catalog-list.proposed.json), [GPU detail](examples/gpu-detail.proposed.json),
  [Storage detail](examples/storage-detail.proposed.json), [motherboard detail](examples/motherboard-detail.proposed.json)
- [Request เดิมที่มี GPU owned และ Storage 3/owned 2](examples/validate.request.proposed.json)
- [HTTP 200 unknown แบบเต็ม](examples/validate-unknown.response.proposed.json): compatible=false,
  issues=[], purchase_quantity Storage=1 แต่ checks ต้องประเมิน 3 instances
- [รูปแบบ passed/failed/unknown ตาม preconditions](examples/check-outcomes.proposed.json)
  ไม่ใช่การประกาศว่า Catalog ที่ metadata=null ตรวจผ่านหรือไม่ผ่านจริง
- [GET list synthetic มี metadata](examples/catalog.synthetic.proposed.json),
  [GPU detail](examples/gpu-detail.synthetic.proposed.json), [Storage detail](examples/storage-detail.synthetic.proposed.json),
  [motherboard detail](examples/motherboard-detail.synthetic.proposed.json): Product เหมือนใน list ทุก field
- [Full response: motherboard checks passed แต่ overall unknown ช่วง B](examples/validate-motherboard-passed.response.proposed.json)
  คู่ [request synthetic](examples/validate.synthetic.request.proposed.json), total=8000
- [Full response failed ช่วง B](examples/validate-failed.response.proposed.json) คู่ [request](examples/validate-failed.request.proposed.json):
  Board B มีแต่ SATA แต่เลือก NVMe → failed แม้ PSU/Case unknown, total=7500
- [Full response metadata missing](examples/validate-metadata-missing.response.proposed.json) คู่ [request](examples/validate-metadata-missing.request.proposed.json):
  GPU B connector=null → unknown พร้อม missing_fields แบบ object, total=8000
- [Full response known unsupported](examples/validate-unsupported.response.proposed.json) คู่ [request](examples/validate-unsupported.request.proposed.json):
  USB มีข้อมูลแต่ phase B ยังไม่ตรวจ → unknown, missing_fields=[], unsupported_fields มี reference, total=6500
- [Full response overall passed — อนาคต C แบบมีเงื่อนไขเท่านั้น](examples/validate-passed.future.response.proposed.json)
  คู่ [request](examples/validate-passed.future.request.proposed.json), total=11500:
  **ไม่สามารถคำนวณ passed นี้จาก Catalog ชุดนี้ได้** เพราะ PSU/Case/system data/installation context และ checker ยังไม่มี
  ใช้ตกลงรูปแบบ JSON/UI เท่านั้น โดยสมมติว่าอนาคต C ตกลง contract และข้อมูลที่เชื่อถือได้ครบจนทุก check ผ่าน
  ไม่สร้าง fields PSU/Case ใหม่ ห้าม server ช่วง A/B emit ตัวอย่างนี้ และไม่ใช่ runnable acceptance fixture

ตัวอย่างเต็มทุกชุดใช้ envelope/Check/FieldReference เดียวกัน; data.compatibility มีเฉพาะโหมด opt-in

| HTTP / code | ใช้เมื่อ |
|---|---|
| 200 / compatible + compatibility.status | ประเมินสำเร็จ ทั้ง passed/failed/unknown |
| 400 / `VALIDATION_ERROR` | กฎ input เดิม: UUID, counts, duplicates, ราคา/specs ที่ client ส่ง, overflow |
| 400 / `UNSUPPORTED_BUILD_SCOPE` (ใหม่) | GPU รวม >1 หรือ Storage รวม >8; เป็น software scope ไม่ใช่ hardware conflict |
| 400 / `UNSUPPORTED_VALIDATION_MODE` (ใหม่) | โหมดที่ server รุ่นใหม่ไม่รองรับ ห้าม ignore แล้วรายงานตรวจครบ |
| 404 / `NOT_FOUND` | อ้าง product/model ไม่พบหรือ inactive ตาม policy เดิม |
| 405 / `METHOD_NOT_ALLOWED` | HTTP method ไม่ตรง พร้อม Allow เดิม |
| 413 / `PAYLOAD_TOO_LARGE` | body >1 MiB ตามเดิม |
| 415 / `UNSUPPORTED_MEDIA_TYPE` | ไม่ส่ง application/json ตามเดิม |
| 500 / `INTERNAL_ERROR` | server/repository เสีย ไม่ใช้แทน metadata ขาด |

คง error envelope `{"error":{"code":"...","message":"..."}}`
เช่นข้อจำกัด software: `{"error":{"code":"UNSUPPORTED_BUILD_SCOPE","message":"โหมดนี้รองรับ GPU รวมไม่เกิน 1 ใบ"}}`
failure issue codes เสนอ: `GPU_CONNECTOR_MISMATCH`, `STORAGE_INTERFACE_MISMATCH`,
`STORAGE_FORM_FACTOR_MISMATCH`, `STORAGE_KEY_MISMATCH`, `MOTHERBOARD_RESOURCE_CONFLICT`,
`PSU_CONNECTOR_UNSUPPORTED`, `PSU_POWER_INSUFFICIENT`, `GPU_CASE_CLEARANCE_EXCEEDED`, `STORAGE_BAY_CAPACITY_EXCEEDED`
unknown reason codes เสนอ: `NOT_IMPLEMENTED`, `METADATA_MISSING`, `CONTEXT_MISSING`,
`MISSING_COMPONENT`, `UNSUPPORTED_DEVICE_CLASS`; ผ่านใช้ MATCH, ไม่ผ่านใช้ INCOMPATIBLE
เช่นคู่ protocol ไม่ตรงอย่างพิสูจน์ได้ => 200/failed ไม่ใช่ 400; metadata SATA/M.2 ไม่ทราบ => 200/unknown ไม่ใช่ 404

## 8. แผนแบ่งช่วงและเกณฑ์ทดสอบรับงาน

### A — Catalog + ราคา

เพิ่ม typed metadata/provenance แบบ nullable โดยไม่เปลี่ยน 6 fields/ram_items/pricing algorithm
ไม่มีการเติม specs GPU/Storage ให้ fixture เดิมจนมีเอกสารตรง SKU; ราคาอย่างเดียวไม่ทำให้ check ผ่าน
โหมดใหม่ส่ง checks GPU/Storage/Motherboard/PSU/Case ที่ยังไม่ทำเป็น unknown
แต่ CPU/RAM ที่ตรวจแล้วพบ conflict ต้องรวม overall เป็น failed ไม่ทับเป็น unknown

- GET list/detail แสดง fields ตามประเภทและค่า null/[] ต่างความหมาย; pagination/active/price ยังเดิม
- ใช้ items เดิมซื้อ GPU/Storage, duplicate row ปฏิเสธ, counts ผิด/overflow เป็น 400
- GPU 0/1 ผ่าน scope, 2 ไม่ผ่าน scope แม้ owned ทั้งหมด; Storage 0/1/8 ผ่าน scope, 9 ไม่ผ่าน scope
- Storage นับ quantity ทุก class รวม USB/SAS/other และ owned; unknown product ยัง 404 ก่อน limit
- Storage quantity=3 owned=2 คิดเงิน 1 ลูก; GPU owned=1 ราคาเพิ่ม 0; RAM selection เดิมไม่เปลี่ยน
- default legacy response/ชนิด fields เดิมไม่เปลี่ยน; extended mode ที่ไม่ echo mode ต้องไม่ถูก FE รับเป็นผลตรวจใหม่
- ราคา/compatibility/specs/evidence ที่ client ส่งไม่ถูกใช้; existing success fixture ยัง total=11500

### B — ตรวจ Motherboard

เริ่มหลังมีข้อมูล manufacturer-reviewed host/profile/ports/sharing ที่พอ ไม่ใช่แค่ SQL columns
เปิดใช้ FE mode ใหม่หลัง UI แยก unknown แล้ว เทียบ resource ของ owned และของที่จะซื้อเท่ากัน

- GPU physical/wired lanes แตกต่าง, ไม่มี slot ที่ยืนยันว่าใช้ได้, slot metadata ไม่ครบ ให้ผลต่างกันถูกต้อง
- M.2 SATA vs PCIe NVMe, key/width/length ตรง/ไม่ตรง, SATA drive บน SATA port และกรณี unknown
- Lower link speed ที่ผู้ผลิตยืนยันให้ warning ไม่ fail; ไม่สมมติ bandwidth/SLI/boot support
- Storage หลายลูกต่อ SKU ใช้จำนวน ports ตาม quantity; owned ใช้ ports ด้วย
- Shared-resource disable/restrict, CPU profile ต่างกัน/ขาด CPU/BIOS ไม่ทราบ และปฏิสัมพันธ์ GPU+Storage
- มี feasible arrangement ที่ greedy พลาดต้องผ่าน; unknown rule ที่อาจเปลี่ยนคำตอบต้อง unknown
- ทดสอบ permutations ของ items ได้ผลคงที่; repository snapshot/field evidence เป็นแหล่งจริง ไม่รับของ client
- motherboard checks ผ่าน แต่ PSU/Case ขาด ต้อง overall unknown ไม่แสดง “ตรวจครบ”
- motherboard failed + PSU/Case unknown ต้อง overall failed; check ที่ผ่านไม่ทำให้ overall ผ่านเอง
- missing_fields/unsupported_fields ใช้ FieldReference[] ทุก status; missing component product_id=null พร้อม category
- known unsupported มี missing_fields=[] ถ้าข้อมูลไม่ขาด; null ที่ไม่ applicable ไม่ถูกนับเป็น missing

### C — ตรวจ PSU/Case

ต้องตกลง data contract แยกของ PSU/Case และบริบทการติดตั้งก่อนเปิด check ใช้ items เดิมได้
แต่รายละเอียดสาย, adapter, BIOS, radiator/cage/spine layout ไม่ได้อยู่ใน request นี้
ถ้าไม่มีทางรู้บริบทจากข้อมูลที่เชื่อถือได้ ให้คง unknown ไม่สมมติ default layout หรือสายที่ผู้ใช้มี

ข้อมูลขั้นต่ำที่ยังขาด: PSU รุ่น/revision/กำลังและข้อกำหนดระบบ, GPU power options+สายที่รองรับ,
CPU/system power envelope รวม startup ของ storage และอุปกรณ์อื่น, case รุ่น/layout/bays,
GPU dimensions+พื้นที่หัวสาย, radiator/drive cages/แผง side clearance [S3](https://www.nvidia.com/content/geforce-gtx/GeForce_RTX_4090_QSG_Rev1.pdf), [S5](https://seasonic.com/seasonic-12v-2x6-cable/), [S6](https://support.fractal-design.com/support/solutions/articles/4000203863)
SYSTEM checks ยังต้องมี CPU/mainboard power connector requirements และ motherboard dimensions/form factor
พร้อมข้อกำหนดอุปกรณ์อื่นที่ติดตั้ง; enum สายไฟของ GPU/Storage ในเอกสารนี้ยังไม่ใช่ PSU schema ทั้งระบบ
ไม่ใช้สูตร TDP GPU+CPU หรือบวก headroom เปอร์เซ็นต์ที่เดาเองรับรอง PSU ทั้งระบบ
min_system_power_w เป็นเงื่อนไขหนึ่ง ไม่แทน connector/cable/peak budget; case length ค่าเดียวก็ไม่พอ

- กำลังวัตต์พอแต่ connector/cable ไม่รองรับต้องไม่ผ่าน; สาย/adapter ไม่ทราบต้อง unknown
- ตรวจทั้งตัวที่ owned และซื้อใหม่; storage startup/bay counts รวม quantity
- การ์ดยาว/สูง/หนาตรง boundary ที่ข้อมูลผู้ผลิตกำหนด; ต้องมี bracket/bay และ cable clearance จริง
- เปลี่ยน radiator/cage/layout แล้วผลเปลี่ยนตาม rule ของ case ไม่ตามชื่อรุ่น GPU [S6](https://support.fractal-design.com/support/solutions/articles/4000203863)
- missing context ต้อง unknown; display VRAM/capacity/ความเร็วสูงต่ำไม่ทำให้ fail
- รวม statuses ถูกต้อง: failed ชนะ unknown; ไม่มี fail แต่ยังมี unknown ไม่เป็น passed
- ทุก check ผ่านก็ยัง build_completeness=not_evaluated, ไม่มีการเปิด publish/save โดยปริยาย

## 9. สิ่งที่ต้องให้ Fluke ตัดสินใจก่อนพัฒนา

1. ยืนยันโครง nested display/host/power/dimensions/provenance, nullable และ enums
2. ยืนยัน GPU สูงสุด 1 / Storage สูงสุด 8 เป็น software limit นับ Storage ทุกชนิดรวม owned;
   phase B ตรวจ SATA+M.2 ก่อน ชนิดอื่นยังนับ limit และคืน unknown หากไม่เกิน limit
3. ยืนยัน opt-in query/ชื่อ mode และ semantics compatible ของโหมดใหม่ พร้อม FE rollout gate
4. ยืนยันชื่อ checks/reason/issue codes, FieldReference ของ missing_fields/unsupported_fields,
   กฎ failed ชนะ unknown, unknown UI และการไม่แสดง full compatibility
5. ระบุ SKU/revision จริงและเอกสารของ GPU/Storage/motherboard ที่จะเริ่ม ingest รวม CPU identity/profile
6. ตกลง policy inactive-owned ภายหลัง, boot/RAID/external storage/multi-GPU ว่ายังอยู่นอก scope
7. ก่อนช่วง C ต้องตกลง PSU/Case fields, power policy และแหล่ง installation context แยกต่างหาก

รายละเอียดหลักฐานและขอบเขตการอนุมานอยู่ใน [manufacturer-rules.md](manufacturer-rules.md)
รอบนี้ตรวจเฉพาะเอกสาร/JSON ไม่รันหรือเพิ่ม tests implementation ใหม่; ตารางด้านบนเป็น acceptance plan

## สรุปพร้อมส่งให้ฟลุค

> ฟลุค ขอ review proposal GPU/Storage ใน backend/proposals/gpu-storage/README.md ก่อนเริ่มเขียนโค้ด
> ยังใช้ items เดิม/ram_items เดิม เพิ่ม Catalog specs ที่แยก display กับ hardware requirements และไม่เดาค่าที่ขาด
> เพิ่ม synthetic Catalog ที่มี metadata พร้อม full responses จับคู่ request แล้ว ดู examples/README.md
> เสนอ GPU ไม่เกิน 1 ใบ/Storage ทุกชนิดรวมไม่เกิน 8 ลูก รวม owned ซึ่งเป็นส่วนหนึ่งของ quantity; หัก owned เฉพาะราคา
> เสนอ opt-in validation_mode=gpu_storage_v1 เพื่อแยก passed/failed/unknown โดย default เดิมไม่เปลี่ยน
> มี failed ที่ยืนยันได้ให้ overall failed แม้ PSU/Case unknown; checks ที่ทำได้ผ่านแต่ยังเหลือ required unknown ให้ overall unknown
> missing_fields/unsupported_fields เป็น object arrays รูปแบบเดียวกัน; ข้อมูลครบส่ง [] และแยก known unsupported ออกจาก missing
> ช่วงแรกทำ Catalog+ราคา ต่อด้วย motherboard แล้วค่อย PSU/Case; ยังไม่อ้างว่าตรวจครบหรือเปิดเผยแพร่ public
